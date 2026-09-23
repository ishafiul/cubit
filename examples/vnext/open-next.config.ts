// OpenNext configuration for Cloudflare Workers
export default {
  default: {
    override: {
      wrapper: 'cloudflare-edge',
      converter: 'edge',
      incrementalCache: 'dummy',
      tagCache: 'dummy',
      queue: 'dummy',
    },
  },
};
