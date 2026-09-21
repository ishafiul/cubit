import { describe, it, expect } from 'vitest';
import {
  router,
  rootRoute,
  indexRoute,
  appsRoute,
  appDetailRoute,
  dynamicWorkersRoute,
  durableObjectsRoute,
  containersRoute,
  kvRoute,
  d1Route,
  r2Route,
  staticAssetsRoute,
  cronRoute,
  queuesRoute,
  workflowsRoute,
  nodesRoute,
  domainsRoute,
  logsRoute,
  githubRoute,
} from './router';

describe('Given the Cubit TanStack Router configuration', () => {
  describe('When inspecting route path definitions', () => {
    it('Then indexRoute targets root path /', () => {
      expect(indexRoute.fullPath).toBe('/');
    });

    it('Then appsRoute targets /apps', () => {
      expect(appsRoute.fullPath).toBe('/apps');
    });

    it('Then appDetailRoute targets /apps/$appId', () => {
      expect(appDetailRoute.fullPath).toBe('/apps/$appId');
    });

    it('Then fleet, settings and routing routes match their expected paths', () => {
      expect(nodesRoute.fullPath).toBe('/nodes');
      expect(domainsRoute.fullPath).toBe('/domains');
      expect(logsRoute.fullPath).toBe('/logs');
      expect(githubRoute.fullPath).toBe('/github');
    });

    it('Then all 10 Cloudflare service routes are declared with exact paths', () => {
      expect(dynamicWorkersRoute.fullPath).toBe('/dynamic-workers');
      expect(durableObjectsRoute.fullPath).toBe('/durable-objects');
      expect(containersRoute.fullPath).toBe('/containers');
      expect(kvRoute.fullPath).toBe('/kv');
      expect(d1Route.fullPath).toBe('/d1');
      expect(r2Route.fullPath).toBe('/r2');
      expect(staticAssetsRoute.fullPath).toBe('/static-assets');
      expect(cronRoute.fullPath).toBe('/cron');
      expect(queuesRoute.fullPath).toBe('/queues');
      expect(workflowsRoute.fullPath).toBe('/workflows');
    });
  });

  describe('When validating route search parameters', () => {
    describe('for /apps/$appId route', () => {
      it('Then valid tab string is preserved', () => {
        const validate = (appDetailRoute.options as any).validateSearch;
        const result = validate({ tab: 'code' });
        expect(result).toEqual({ tab: 'code' });
      });

      it('Then non-string or missing tab becomes undefined', () => {
        const validate = (appDetailRoute.options as any).validateSearch;
        expect(validate({})).toEqual({ tab: undefined });
        expect(validate({ tab: 123 })).toEqual({ tab: undefined });
      });
    });

    describe('for service routes with resource id search params', () => {
      it('Then valid string id is extracted on /kv', () => {
        const validate = (kvRoute.options as any).validateSearch;
        expect(validate({ id: 'ns-user-sessions' })).toEqual({ id: 'ns-user-sessions' });
      });

      it('Then valid string id is extracted on /d1', () => {
        const validate = (d1Route.options as any).validateSearch;
        expect(validate({ id: 'db-prod-analytics' })).toEqual({ id: 'db-prod-analytics' });
      });

      it('Then valid string id is extracted on /r2', () => {
        const validate = (r2Route.options as any).validateSearch;
        expect(validate({ id: 'bucket-media-assets' })).toEqual({ id: 'bucket-media-assets' });
      });

      it('Then missing or non-string id safely returns undefined', () => {
        const validate = (kvRoute.options as any).validateSearch;
        expect(validate({})).toEqual({ id: undefined });
        expect(validate({ id: null })).toEqual({ id: undefined });
      });
    });

    describe('for /logs route', () => {
      it('Then valid appId string is parsed', () => {
        const validate = (logsRoute.options as any).validateSearch;
        expect(validate({ appId: 'app-123' })).toEqual({ appId: 'app-123' });
      });

      it('Then missing appId returns undefined', () => {
        const validate = (logsRoute.options as any).validateSearch;
        expect(validate({})).toEqual({ appId: undefined });
      });
    });
  });

  describe('When inspecting the constructed router instance', () => {
    it('Then router contains rootRoute and configured routeTree', () => {
      expect(router).toBeDefined();
      expect(router.routeTree).toBeDefined();
      expect(rootRoute.children).toBeDefined();
      expect((rootRoute.children as any)?.length).toBe(17);
    });
  });
});
