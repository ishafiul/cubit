/**
 * Utility to dynamically resolve worker subdomain URLs based on the active host and port.
 */

export function getActivePort(fallback = 8000): number {
  if (typeof window !== 'undefined' && window.location && window.location.port) {
    const parsed = parseInt(window.location.port, 10);
    if (!isNaN(parsed) && parsed > 0) return parsed;
  }
  return fallback;
}

export function getSubdomainUrl(subdomain: string, preferredUrl?: string): string {
  if (typeof window !== 'undefined' && window.location) {
    const currentPort = window.location.port;
    const portStr = currentPort ? `:${currentPort}` : '';
    const protocol = window.location.protocol || 'http:';
    let hostname = window.location.hostname || 'localhost';
    if (hostname === '127.0.0.1') hostname = 'localhost';

    // If preferredUrl was provided, check if it needs port adjustment for local environment
    if (preferredUrl) {
      try {
        const u = new URL(preferredUrl);
        if (
          (u.hostname === 'localhost' || u.hostname.endsWith('.localhost') || u.hostname === '127.0.0.1') &&
          (hostname === 'localhost' || hostname.endsWith('.localhost'))
        ) {
          if (currentPort && u.port !== currentPort) {
            u.port = currentPort;
            return u.toString().replace(/\/$/, '');
          }
        }
        return preferredUrl.replace(/\/$/, '');
      } catch {
        // Fall through to dynamic URL generation
      }
    }

    const cleanSubdomain = (subdomain || 'worker').trim();
    return `${protocol}//${cleanSubdomain}.${hostname}${portStr}`;
  }

  return preferredUrl || `http://${subdomain || 'worker'}.localhost:8000`;
}
