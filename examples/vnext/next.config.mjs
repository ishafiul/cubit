/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Cloudflare Workers Edge runtime support
  experimental: {
    runtime: 'edge',
  },
};

export default nextConfig;
