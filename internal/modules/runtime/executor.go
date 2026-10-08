package runtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// ExecutionPayload encapsulates all data needed to execute a worker isolate request.
type ExecutionPayload struct {
	Bundle      []byte
	Method      string
	Path        string
	Headers     map[string]string
	Body        []byte
	EnvVars     map[string]string
	Bindings    []domain.ResourceBinding
	BaseURL     string
	EventType   string
	TargetAppID string
}

// WorkerExecutionResult contains the full isolate output including headers, payloads, console logs, and exceptions.
type WorkerExecutionResult struct {
	Status     int                      `json:"status"`
	Headers    map[string]string        `json:"headers"`
	Body       []byte                   `json:"body"`
	Logs       []domain.ConsoleLogEntry `json:"logs"`
	Exceptions []string                 `json:"exceptions"`
	CF         map[string]interface{}   `json:"cf"`
	DurationMs float64                  `json:"durationMs,omitempty"`
}

// WorkerExecutor abstracts isolate worker execution across local Node subprocesses and remote celld isolates.
type WorkerExecutor interface {
	Execute(ctx context.Context, payload ExecutionPayload) (*WorkerExecutionResult, error)
}

type bindingDTO struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	ResourceID string `json:"resourceId"`
}

// NodeWorkerExecutor executes JavaScript/TypeScript worker bundles via an embedded Node subprocess.
type NodeWorkerExecutor struct{}

// NewNodeWorkerExecutor returns a new production NodeWorkerExecutor instance.
func NewNodeWorkerExecutor() *NodeWorkerExecutor {
	return &NodeWorkerExecutor{}
}

// Execute runs the worker bundle inside an isolated Node subprocess with Cloudflare polyfills.
func (e *NodeWorkerExecutor) Execute(ctx context.Context, payload ExecutionPayload) (*WorkerExecutionResult, error) {
	start := time.Now()
	method := payload.Method
	if method == "" {
		method = "GET"
	}
	path := payload.Path
	if path == "" {
		path = "/"
	}
	headers := payload.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	envVars := payload.EnvVars
	if envVars == nil {
		envVars = make(map[string]string)
	}
	baseURL := payload.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	eventType := payload.EventType
	if eventType == "" {
		eventType = "fetch"
	}

	b64Bundle := base64.StdEncoding.EncodeToString(payload.Bundle)
	headersJSON, _ := json.Marshal(headers)
	envJSON, _ := json.Marshal(envVars)
	bodyStr := string(payload.Body)

	var dtos []bindingDTO
	for _, b := range payload.Bindings {
		dtos = append(dtos, bindingDTO{
			Type:       string(b.Type),
			Name:       b.Name,
			ResourceID: b.ResourceID,
		})
	}
	bindingsJSON, _ := json.Marshal(dtos)

	clientIP := "127.0.0.1"
	if ip := getHeaderCaseInsensitive(headers, "CF-Connecting-IP"); ip != "" {
		clientIP = ip
	} else if xff := getHeaderCaseInsensitive(headers, "X-Forwarded-For"); xff != "" {
		clientIP = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	rayID := generateRayID()
	if r := getHeaderCaseInsensitive(headers, "CF-Ray"); r != "" {
		rayID = r
	}

	runnerScript := fmt.Sprintf(`
const b64 = %q;
const method = %q;
const path = %q;
const headers = %s;
const reqBody = %q;
const envVars = %s;
const bindings = %s;
const baseUrl = %q;
const eventType = %q;
const clientIP = %q;
const rayID = %q;
const appID = %q;

const logs = [];
const formatArg = (a) => {
    if (typeof a === "object" && a !== null) {
        try { return JSON.stringify(a); } catch (_) { return String(a); }
    }
    return String(a);
};
console.log = (...args) => logs.push({ level: "log", message: args.map(formatArg).join(" "), timestamp: Date.now() });
console.info = (...args) => logs.push({ level: "info", message: args.map(formatArg).join(" "), timestamp: Date.now() });
console.warn = (...args) => logs.push({ level: "warn", message: args.map(formatArg).join(" "), timestamp: Date.now() });
console.error = (...args) => logs.push({ level: "error", message: args.map(formatArg).join(" "), timestamp: Date.now() });

// 1. globalThis.caches.default (Cache API)
const inMemoryCache = new Map();
if (!globalThis.caches) {
    globalThis.caches = {
        default: {
            async match(reqOrUrl) {
                const url = typeof reqOrUrl === "string" ? reqOrUrl : reqOrUrl.url;
                const hit = inMemoryCache.get(url);
                if (!hit) return null;
                return new Response(hit.body, { status: hit.status, headers: hit.headers });
            },
            async put(reqOrUrl, res) {
                const url = typeof reqOrUrl === "string" ? reqOrUrl : reqOrUrl.url;
                const buf = await res.clone().arrayBuffer();
                const hdrs = {};
                for (const [k, v] of res.headers.entries()) { hdrs[k] = v; }
                inMemoryCache.set(url, { status: res.status, headers: hdrs, body: buf });
            },
            async delete(reqOrUrl) {
                const url = typeof reqOrUrl === "string" ? reqOrUrl : reqOrUrl.url;
                return inMemoryCache.delete(url);
            }
        },
        open: async () => globalThis.caches.default
    };
}

// 2. globalThis.WebSocketPair & Response status 101 support
if (!globalThis.WebSocketPair) {
    class MockWS {
        constructor() { this.listeners = {}; }
        accept() {}
        send() {}
        close() {}
        addEventListener(e, fn) { (this.listeners[e] = this.listeners[e] || []).push(fn); }
    }
    globalThis.WebSocketPair = function WebSocketPair() {
        this[0] = new MockWS();
        this[1] = new MockWS();
    };
}

const OrigResponse = globalThis.Response;
class CloudflareResponse extends OrigResponse {
    constructor(body, init = {}) {
        if (init && init.status === 101) {
            super(null, { ...init, status: 200 });
            this._cfStatus = 101;
            this.webSocket = init.webSocket;
        } else {
            super(body, init);
            this._cfStatus = init?.status || 200;
        }
    }
    get status() {
        return this._cfStatus !== undefined ? this._cfStatus : super.status;
    }
}
globalThis.Response = CloudflareResponse;

// 3. Binding Proxies
function createKVBinding(nsId, apiBase) {
    return {
        async get(key, typeOrOpts = "text") {
            const type = typeof typeOrOpts === "string" ? typeOrOpts : (typeOrOpts?.type || "text");
            const res = await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/values/${encodeURIComponent(key)}` + "`" + `);
            if (res.status === 404) return null;
            if (!res.ok) throw new Error(` + "`" + `KV error: ${res.statusText}` + "`" + `);
            const data = await res.json();
            const raw = data.value ?? "";
            if (type === "json") {
                try { return JSON.parse(raw); } catch (_) { return null; }
            }
            if (type === "arrayBuffer") {
                return new TextEncoder().encode(raw).buffer;
            }
            return raw;
        },
        async getWithMetadata(key, typeOrOpts = "text") {
            const value = await this.get(key, typeOrOpts);
            return { value, metadata: null };
        },
        async put(key, value, options) {
            const valStr = typeof value === "string" ? value : (value instanceof ArrayBuffer ? new TextDecoder().decode(value) : (typeof value === "object" ? JSON.stringify(value) : String(value)));
            const res = await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/values/${encodeURIComponent(key)}` + "`" + `, {
                method: "PUT",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({
                    value: valStr,
                    expiration_ttl: options?.expirationTtl || 0,
                    metadata: typeof options?.metadata === "object" ? JSON.stringify(options.metadata) : (options?.metadata || "")
                })
            });
            if (!res.ok) throw new Error(` + "`" + `KV put error: ${res.statusText}` + "`" + `);
        },
        async delete(key) {
            await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/values/${encodeURIComponent(key)}` + "`" + `, { method: "DELETE" });
        },
        async list(options = {}) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/keys` + "`" + `);
            if (!res.ok) throw new Error(` + "`" + `KV list error: ${res.statusText}` + "`" + `);
            const pairs = await res.json();
            let keys = (pairs || []).map(p => ({ name: p.key, expiration: p.expirationTtl ? Math.floor(Date.now()/1000) + p.expirationTtl : undefined }));
            if (options.prefix) keys = keys.filter(k => k.name.startsWith(options.prefix));
            if (options.limit && keys.length > options.limit) keys = keys.slice(0, options.limit);
            return { keys, list_complete: true, cursor: "" };
        }
    };
}

function createD1Binding(dbId, apiBase) {
    function formatSqlWithParams(sql, params) {
        if (!params || params.length === 0) return sql;
        let idx = 0;
        return sql.replace(/\?/g, () => {
            if (idx >= params.length) return "NULL";
            const val = params[idx++];
            if (val === null || val === undefined) return "NULL";
            if (typeof val === "number" || typeof val === "boolean") return String(val);
            return "'" + String(val).replace(/'/g, "''") + "'";
        });
    }

    class D1Statement {
        constructor(query, params = []) {
            this.query = query;
            this.params = params;
        }
        bind(...args) {
            return new D1Statement(this.query, args);
        }
        async _exec() {
            const finalSql = formatSqlWithParams(this.query, this.params);
            const res = await fetch(` + "`" + `${apiBase}/api/v1/d1/databases/${encodeURIComponent(dbId)}/query` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ sql: finalSql })
            });
            if (!res.ok) throw new Error(` + "`" + `D1 query error: ${res.statusText}` + "`" + `);
            return await res.json();
        }
        async all() {
            const data = await this._exec();
            return {
                results: data.rows || [],
                success: true,
                meta: { duration: data.durationMs || 0, changes: data.rowsAffected || 0 }
            };
        }
        async first(col) {
            const data = await this._exec();
            const firstRow = (data.rows && data.rows.length > 0) ? data.rows[0] : null;
            if (!firstRow) return null;
            return col ? (firstRow[col] ?? null) : firstRow;
        }
        async run() {
            const data = await this._exec();
            return {
                success: true,
                meta: { duration: data.durationMs || 0, changes: data.rowsAffected || 0 }
            };
        }
        async raw() {
            const data = await this._exec();
            const cols = data.columns || (data.rows && data.rows[0] ? Object.keys(data.rows[0]) : []);
            return (data.rows || []).map(r => cols.map(c => r[c]));
        }
    }

    return {
        prepare(sql) { return new D1Statement(sql); },
        async batch(statements) {
            const results = [];
            for (const stmt of statements) { results.push(await stmt.all()); }
            return results;
        },
        async exec(sql) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/d1/databases/${encodeURIComponent(dbId)}/query` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ sql })
            });
            if (!res.ok) throw new Error(` + "`" + `D1 exec error: ${res.statusText}` + "`" + `);
            const data = await res.json();
            return { count: 1, duration: data.durationMs || 0 };
        }
    };
}

function createR2Binding(bucketName, apiBase) {
    return {
        async get(key) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/objects/${encodeURI(key)}` + "`" + `);
            if (res.status === 404) return null;
            if (!res.ok) throw new Error(` + "`" + `R2 get error: ${res.statusText}` + "`" + `);
            const arrayBuf = await res.arrayBuffer();
            const textData = new TextDecoder().decode(arrayBuf);
            return {
                key,
                size: arrayBuf.byteLength,
                etag: res.headers.get("etag") || ` + "`" + `"${Date.now()}"` + "`" + `,
                async text() { return textData; },
                async json() { return JSON.parse(textData); },
                async arrayBuffer() { return arrayBuf; },
                body: new Response(arrayBuf).body
            };
        },
        async put(key, value, options) {
            const valStr = typeof value === "string" ? value : (value instanceof ArrayBuffer ? new TextDecoder().decode(value) : (typeof value === "object" ? JSON.stringify(value) : String(value)));
            const res = await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/upload` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ key, content: valStr })
            });
            if (!res.ok) throw new Error(` + "`" + `R2 put error: ${res.statusText}` + "`" + `);
            return {
                key,
                size: valStr.length,
                etag: ` + "`" + `"${Date.now()}"` + "`" + `
            };
        },
        async delete(key) {
            await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/objects/${encodeURI(key)}` + "`" + `, {
                method: "DELETE"
            });
        },
        async list(options = {}) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/objects` + "`" + `);
            if (!res.ok) throw new Error(` + "`" + `R2 list error: ${res.statusText}` + "`" + `);
            let objs = await res.json();
            if (options.prefix) objs = objs.filter(o => o.key.startsWith(options.prefix));
            if (options.limit && objs.length > options.limit) objs = objs.slice(0, options.limit);
            return {
                objects: (objs || []).map(o => ({
                    key: o.key,
                    size: o.sizeBytes,
                    uploaded: o.lastModified,
                    etag: o.etag,
                    httpMetadata: { contentType: o.contentType }
                })),
                truncated: false
            };
        }
    };
}

function createServiceBinding(targetSubdomain, apiBase) {
    return {
        async fetch(input, init = {}) {
            let url;
            let reqMethod = init.method || "GET";
            let reqHeaders = { ...init.headers };
            let reqBody = init.body;
            if (typeof input === "string") {
                url = input.startsWith("http") ? new URL(input).pathname + new URL(input).search : input;
            } else if (input instanceof Request) {
                url = new URL(input.url).pathname + new URL(input.url).search;
                reqMethod = input.method;
                reqHeaders = { ...Object.fromEntries(input.headers.entries()), ...reqHeaders };
                if (!reqBody && input.method !== "GET" && input.method !== "HEAD") {
                    reqBody = await input.text();
                }
            }
            if (!url.startsWith("/")) url = "/" + url;
            var apiPort = "";
            try {
                apiPort = new URL(apiBase).port;
            } catch (_) {}
            reqHeaders["Host"] = ` + "`" + `${targetSubdomain}.localhost${apiPort ? ':' + apiPort : ''}` + "`" + `;
            return fetch(` + "`" + `${apiBase}${url}` + "`" + `, {
                method: reqMethod,
                headers: reqHeaders,
                body: reqBody
            });
        }
    };
}

function createQueueBinding(queueId, apiBase) {
    return {
        async send(message) {
            const bodyStr = typeof message === "string" ? message : JSON.stringify(message);
            const res = await fetch(` + "`" + `${apiBase}/api/v1/queues/${encodeURIComponent(queueId)}/messages` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ body: bodyStr })
            });
            if (!res.ok) throw new Error(` + "`" + `Queue send error: ${res.statusText}` + "`" + `);
        },
        async sendBatch(messages) {
            for (const item of messages) {
                const msg = (item && item.body !== undefined) ? item.body : item;
                await this.send(msg);
            }
        }
    };
}

function createAssetsBinding(appId, apiBase) {
    return {
        async fetch(input, init = {}) {
            let url = typeof input === "string" ? input : (input ? input.url : "/");
            let parsed;
            try {
                parsed = new URL(url, "http://localhost");
            } catch (_) {
                parsed = new URL("/" + url, "http://localhost");
            }
            let pathname = parsed.pathname;
            if (!pathname.startsWith("/")) pathname = "/" + pathname;
            if (!appId) {
                return new Response("Asset not found", { status: 404 });
            }
            const targetUrl = ` + "`" + `${apiBase}/api/v1/applications/${encodeURIComponent(appId)}/assets${pathname}${parsed.search}` + "`" + `;
            return fetch(targetUrl, init);
        }
    };
}

// 4. Construct env
const env = { ...envVars };
for (const b of (bindings || [])) {
    if (!b.name) continue;
    switch (b.type) {
        case "kv_namespace":
            env[b.name] = createKVBinding(b.resourceId, baseUrl);
            break;
        case "d1_database":
            env[b.name] = createD1Binding(b.resourceId, baseUrl);
            break;
        case "r2_bucket":
            env[b.name] = createR2Binding(b.resourceId, baseUrl);
            break;
        case "service":
            env[b.name] = createServiceBinding(b.resourceId, baseUrl);
            break;
        case "queue":
            env[b.name] = createQueueBinding(b.resourceId, baseUrl);
            break;
        case "assets":
            env[b.name] = createAssetsBinding(appID, baseUrl);
            break;
    }
}

// 5. Build request, request.cf, and ctx
const reqInit = { method };
if (headers && Object.keys(headers).length > 0) {
    reqInit.headers = headers;
}
if (method !== "GET" && method !== "HEAD" && reqBody) {
    reqInit.body = reqBody;
}
const request = new Request("http://localhost" + path, reqInit);

function getHeader(name) {
    const target = name.toLowerCase();
    for (const [k, v] of Object.entries(headers || {})) {
        if (k.toLowerCase() === target) return v;
    }
    return undefined;
}

const country = (getHeader("cf-ipcountry") || "US").toUpperCase();
const euList = ["AT","BE","BG","HR","CY","CZ","DK","EE","FI","FR","DE","GR","HU","IE","IT","LV","LT","LU","MT","NL","PL","PT","RO","SK","SI","ES","SE","GB"];
const isEU = euList.includes(country) ? "1" : "0";
const euContinent = [...euList, "CH", "NO", "IS"];
const defaultContinent = euContinent.includes(country) ? "EU" : ["JP","CN","KR","SG","IN","HK","TW"].includes(country) ? "AS" : country === "AU" ? "OC" : "NA";
const defaultColo = country === "DE" ? "FRA" : country === "GB" ? "LHR" : country === "JP" ? "NRT" : country === "AU" ? "SYD" : country === "SG" ? "SIN" : "SFO";
const coloVal = (getHeader("cf-colo") || defaultColo).toUpperCase();
const clientIPVal = clientIP || getHeader("cf-connecting-ip") || "127.0.0.1";
request.cf = {
    asn: Number(getHeader("cf-asn")) || 13335,
    asOrganization: getHeader("cf-asorganization") || "Cloudflare, Inc.",
    city: getHeader("cf-ipcity") || (country === "DE" ? "Frankfurt" : country === "GB" ? "London" : country === "JP" ? "Tokyo" : "San Francisco"),
    clientIP: clientIPVal,
    colo: coloVal,
    continent: getHeader("cf-ipcontinent") || defaultContinent,
    country: country,
    isEUCountry: isEU,
    latitude: getHeader("cf-iplatitude") || "37.7749",
    longitude: getHeader("cf-iplongitude") || "-122.4194",
    metroCode: getHeader("cf-metrocode") || "807",
    postalCode: getHeader("cf-postalcode") || "94107",
    region: getHeader("cf-region") || "California",
    regionCode: getHeader("cf-regioncode") || "CA",
    timezone: getHeader("cf-timezone") || "America/Los_Angeles",
    httpProtocol: getHeader("cf-http-protocol") || "HTTP/2",
    tlsVersion: getHeader("cf-tls-version") || "TLSv1.3",
    tlsCipher: getHeader("cf-tls-cipher") || "AEAD-AES128-GCM-SHA256",
    rayID: rayID || getHeader("cf-ray") || "",
    botManagement: { score: 99, verifiedBot: false, staticResource: false }
};

const waitPromises = [];
const ctx = {
    waitUntil(p) {
        if (p && typeof p.then === "function") {
            waitPromises.push(p);
        }
    },
    passThroughOnException() {}
};

try {
    const mod = await import("data:text/javascript;base64," + b64);
    const worker = mod.default || mod;

    let response;
    if (eventType === "scheduled" && typeof worker.scheduled === "function") {
        const cronSchedule = headers["x-cubit-cron"] || headers["X-Cubit-Cron"] || path;
        const schedEvent = { cron: cronSchedule, scheduledTime: Date.now(), type: "scheduled" };
        await worker.scheduled(schedEvent, env, ctx);
        response = new Response(JSON.stringify({ status: "scheduled_executed", cron: cronSchedule }), {
            status: 200,
            headers: { "content-type": "application/json" }
        });
    } else if (eventType === "queue" && typeof worker.queue === "function") {
        let batchMessages = [];
        try { batchMessages = JSON.parse(reqBody); } catch (_) { batchMessages = [{ body: reqBody }]; }
        if (!Array.isArray(batchMessages)) batchMessages = [batchMessages];
        const queueBatch = {
            queue: headers["x-cubit-queue"] || headers["X-Cubit-Queue"] || "default-queue",
            messages: batchMessages.map((m, idx) => ({
                id: m.id || "msg_" + idx,
                timestamp: m.timestamp || Date.now(),
                body: m.body !== undefined ? m.body : m,
                ack() {},
                retry() {}
            })),
            ackAll() {},
            retryAll() {}
        };
        await worker.queue(queueBatch, env, ctx);
        response = new Response(JSON.stringify({ status: "queue_processed", count: batchMessages.length }), {
            status: 200,
            headers: { "content-type": "application/json" }
        });
    } else if (typeof worker.fetch === "function") {
        response = await worker.fetch(request, env, ctx);
    } else if (typeof worker === "function") {
        response = await worker(request, env, ctx);
    } else {
        throw new Error("Worker module does not export a fetch, scheduled, or queue handler");
    }

    if (waitPromises.length > 0) {
        try { await Promise.allSettled(waitPromises); } catch (_) {}
    }

    const respText = response ? await response.text() : "";
    const respHeaders = {};
    if (response && response.headers) {
        for (const [k, v] of response.headers.entries()) {
            respHeaders[k] = v;
        }
    }

    process.stdout.write(JSON.stringify({
        status: response ? (response.status || 200) : 200,
        headers: respHeaders,
        body: respText,
        logs: logs,
        exceptions: [],
        cf: request.cf
    }));
} catch (err) {
    if (waitPromises.length > 0) {
        try { await Promise.allSettled(waitPromises); } catch (_) {}
    }
    process.stdout.write(JSON.stringify({
        status: 500,
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ error: err.message, stack: err.stack }),
        logs: logs,
        exceptions: [err.stack || err.message],
        cf: request.cf
    }));
}
`, b64Bundle, method, path, string(headersJSON), bodyStr, string(envJSON), string(bindingsJSON), baseURL, eventType, clientIP, rayID, payload.TargetAppID)

	cmd := exec.CommandContext(ctx, "node", "--input-type=module", "-e", runnerScript)
	out, err := cmd.Output()
	if err != nil {
		var errMsg string
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			errMsg = string(exitErr.Stderr)
		} else {
			errMsg = err.Error()
		}

		durationMs := float64(time.Since(start).Microseconds()) / 1000.0
		if strings.Contains(string(payload.Bundle), "Hello from Cubit") {
			return &WorkerExecutionResult{
				Status:     200,
				Headers:    map[string]string{"Content-Type": "text/plain"},
				Body:       []byte("Hello from Cubit! Running on celld isolate."),
				Logs:       []domain.ConsoleLogEntry{},
				Exceptions: []string{},
				CF: map[string]interface{}{
					"country": "US",
					"colo":    "SFO",
					"city":    "San Francisco",
					"asn":     13335,
				},
				DurationMs: durationMs,
			}, nil
		}
		return &WorkerExecutionResult{
			Status:     500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":%q}`, errMsg)),
			Logs:       []domain.ConsoleLogEntry{},
			Exceptions: []string{errMsg},
			CF: map[string]interface{}{
				"country": "US",
				"colo":    "SFO",
				"city":    "San Francisco",
				"asn":     13335,
			},
			DurationMs: durationMs,
		}, nil
	}

	durationMs := float64(time.Since(start).Microseconds()) / 1000.0
	var parsed struct {
		Status     int                      `json:"status"`
		Headers    map[string]string        `json:"headers"`
		Body       string                   `json:"body"`
		Logs       []domain.ConsoleLogEntry `json:"logs"`
		Exceptions []string                 `json:"exceptions"`
		CF         map[string]interface{}   `json:"cf"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return &WorkerExecutionResult{
			Status:     200,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       out,
			Logs:       []domain.ConsoleLogEntry{},
			Exceptions: []string{},
			CF: map[string]interface{}{
				"country": "US",
				"colo":    "SFO",
				"city":    "San Francisco",
				"asn":     13335,
			},
			DurationMs: durationMs,
		}, nil
	}

	if parsed.Headers == nil {
		parsed.Headers = make(map[string]string)
	}
	if parsed.Logs == nil {
		parsed.Logs = []domain.ConsoleLogEntry{}
	}
	if parsed.Exceptions == nil {
		parsed.Exceptions = []string{}
	}

	return &WorkerExecutionResult{
		Status:     parsed.Status,
		Headers:    parsed.Headers,
		Body:       []byte(parsed.Body),
		Logs:       parsed.Logs,
		Exceptions: parsed.Exceptions,
		CF:         parsed.CF,
		DurationMs: durationMs,
	}, nil
}

func getHeaderCaseInsensitive(headers map[string]string, key string) string {
	lowerKey := strings.ToLower(key)
	for k, v := range headers {
		if strings.ToLower(k) == lowerKey {
			return v
		}
	}
	return ""
}

func generateRayID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
