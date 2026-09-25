package deployment

import (
	"bytes"
	"regexp"
)

var (
	esbuildExportDefaultRegex = regexp.MustCompile(`export\s*\{\s*([a-zA-Z0-9_$]+)\s*as\s*default\s*\};?`)
	exportDefaultRegex        = regexp.MustCompile(`export\s+default\s+`)
)

const edgeContextShimHeader = `// --- Cubit Runtime Context Forwarding Shim ---
function __cubit_synthesize_cf(req) {
  if (req.cf && typeof req.cf === "object") return req.cf;
  const h = (k) => (req.headers && typeof req.headers.get === "function" ? req.headers.get(k) : undefined) || undefined;
  const country = (h("cf-ipcountry") || "XX").toUpperCase();
  const euList = ["AT","BE","BG","HR","CY","CZ","DK","EE","FI","FR","DE","GR","HU","IE","IT","LV","LT","LU","MT","NL","PL","PT","RO","SK","SI","ES","SE","GB"];
  const isEU = euList.includes(country) ? "1" : "0";
  const euContinent = [...euList, "CH", "NO", "IS"];
  const continent = euContinent.includes(country) ? "EU" : ["JP","CN","KR","SG","IN","HK","TW"].includes(country) ? "AS" : country === "AU" ? "OC" : "NA";
  const defaultColo = country === "DE" ? "FRA" : country === "GB" ? "LHR" : country === "JP" ? "NRT" : country === "AU" ? "SYD" : country === "SG" ? "SIN" : "CUBIT";

  const cf = {
    asn: Number(h("cf-asn")) || 13335,
    asOrganization: h("cf-asorganization") || "Cloudflare, Inc.",
    city: h("cf-ipcity") || "Default",
    clientIP: h("cf-connecting-ip") || "127.0.0.1",
    colo: (h("cf-colo") || defaultColo).toUpperCase(),
    continent: h("cf-ipcontinent") || continent,
    country: country,
    isEUCountry: isEU,
    latitude: h("cf-iplatitude") || "37.7749",
    longitude: h("cf-iplongitude") || "-122.4194",
    metroCode: h("cf-metrocode") || "807",
    postalCode: h("cf-postalcode") || "94107",
    region: h("cf-region") || "California",
    regionCode: h("cf-regioncode") || "CA",
    timezone: h("cf-timezone") || "America/Los_Angeles",
    httpProtocol: h("cf-http-protocol") || "HTTP/2",
    tlsVersion: h("cf-tls-version") || "TLSv1.3",
    tlsCipher: h("cf-tls-cipher") || "AEAD-AES128-GCM-SHA256",
    rayID: h("cf-ray") || "",
    botManagement: { score: 99, verifiedBot: false, staticResource: false }
  };

  try {
    Object.defineProperty(req, 'cf', { value: cf, writable: true, enumerable: true, configurable: true });
  } catch (_) {
    req.cf = cf;
  }
  return cf;
}
`

const edgeContextShimFooter = `
// --- Cubit Edge Context Proxy Wrapper ---
const __cubit_wrapped_worker__ = new Proxy(typeof __cubit_inner_worker__ !== "undefined" ? __cubit_inner_worker__ : {}, {
  get(target, prop, receiver) {
    if (prop === "fetch") {
      return function(request, env, ctx) {
        if (request) __cubit_synthesize_cf(request);
        const originalFetch = Reflect.get(target, "fetch", receiver);
        if (typeof originalFetch === "function") {
          return originalFetch.call(target, request, env, ctx);
        }
        if (typeof target === "function") {
          return target(request, env, ctx);
        }
      };
    }
    return Reflect.get(target, prop, receiver);
  },
  apply(target, thisArg, argArray) {
    if (argArray && argArray.length > 0) {
      __cubit_synthesize_cf(argArray[0]);
    }
    return Reflect.apply(target, thisArg, argArray);
  }
});
export default __cubit_wrapped_worker__;
`

// WrapBundleWithEdgeContext injects runtime Cloudflare edge context synthesis into a worker bundle.
// When celld or V8 executes worker.fetch(request, env, ctx), request.cf is guaranteed to be populated
// from inbound CF-* edge headers.
func WrapBundleWithEdgeContext(bundle []byte) []byte {
	if bytes.Contains(bundle, []byte("__cubit_synthesize_cf")) {
		return bundle
	}

	bundleStr := string(bundle)
	var transformed string

	if esbuildExportDefaultRegex.MatchString(bundleStr) {
		transformed = esbuildExportDefaultRegex.ReplaceAllString(bundleStr, "const __cubit_inner_worker__ = $1;")
	} else if exportDefaultRegex.MatchString(bundleStr) {
		transformed = exportDefaultRegex.ReplaceAllString(bundleStr, "const __cubit_inner_worker__ = ")
	} else {
		transformed = bundleStr + "\nconst __cubit_inner_worker__ = typeof default !== 'undefined' ? default : {};"
	}

	var buf bytes.Buffer
	buf.WriteString(edgeContextShimHeader)
	buf.WriteString("\n")
	buf.WriteString(transformed)
	buf.WriteString("\n")
	buf.WriteString(edgeContextShimFooter)

	return buf.Bytes()
}
