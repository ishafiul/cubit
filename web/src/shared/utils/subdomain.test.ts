import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { getActivePort, getSubdomainUrl } from './subdomain';

describe('Given getActivePort utility', () => {
  it('When window.location.port is set Then returns the parsed port number', () => {
    Object.defineProperty(window, 'location', {
      value: { port: '9400', hostname: 'localhost', protocol: 'http:' },
      writable: true,
      configurable: true,
    });

    expect(getActivePort()).toBe(9400);
  });

  it('When window.location.port is empty Then returns default fallback', () => {
    Object.defineProperty(window, 'location', {
      value: { port: '', hostname: 'localhost', protocol: 'http:' },
      writable: true,
      configurable: true,
    });

    expect(getActivePort(8000)).toBe(8000);
  });
});

describe('Given getSubdomainUrl utility', () => {
  const originalLocation = window.location;

  beforeEach(() => {
    Object.defineProperty(window, 'location', {
      value: {
        port: '9400',
        hostname: 'localhost',
        protocol: 'http:',
        host: 'localhost:9400',
        origin: 'http://localhost:9400',
      },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    });
  });

  describe('When generating subdomain URL without preferredUrl', () => {
    it('Then generates URL using current port and localhost subdomain', () => {
      const url = getSubdomainUrl('my-api');
      expect(url).toBe('http://my-api.localhost:9400');
    });
  });

  describe('When preferredUrl contains stale port 8000 and current window port is 9400', () => {
    it('Then dynamically updates the port to match current port 9400', () => {
      const url = getSubdomainUrl('my-api', 'http://my-api.localhost:8000');
      expect(url).toBe('http://my-api.localhost:9400');
    });
  });

  describe('When preferredUrl is already matching or on a custom domain', () => {
    it('Then preserves custom external domain intact', () => {
      const url = getSubdomainUrl('my-api', 'https://api.example.com');
      expect(url).toBe('https://api.example.com');
    });
  });
});
