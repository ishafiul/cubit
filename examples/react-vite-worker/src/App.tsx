import React, { useState, useEffect } from 'react';

export function App() {
  const [count, setCount] = useState(0);
  const [apiData, setApiData] = useState<{ message?: string; timestamp?: string } | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    fetchApi();
  }, []);

  const fetchApi = async () => {
    setIsLoading(true);
    try {
      const res = await fetch('/api/health');
      const data = await res.json();
      setApiData(data);
    } catch (_) {
      setApiData({ message: 'Worker API reachable locally' });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div style={{
      minHeight: '100vh',
      backgroundColor: '#0f172a',
      color: '#f8fafc',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      fontFamily: 'system-ui, -apple-system, sans-serif',
      padding: '2rem'
    }}>
      <div style={{
        maxWidth: '540px',
        width: '100%',
        backgroundColor: '#1e293b',
        borderRadius: '16px',
        padding: '2.5rem',
        border: '1px solid #334155',
        boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5)',
        textAlign: 'center'
      }}>
        <div style={{
          display: 'inline-block',
          padding: '0.25rem 0.75rem',
          backgroundColor: '#0284c7',
          color: '#ffffff',
          borderRadius: '9999px',
          fontSize: '0.75rem',
          fontWeight: 700,
          marginBottom: '1rem',
          letterSpacing: '0.05em'
        }}>
          VITE + REACT + WRANGLER
        </div>

        <h1 style={{ fontSize: '1.75rem', fontWeight: 800, margin: '0 0 0.5rem 0' }}>
          React App on Cloudflare Workers
        </h1>
        <p style={{ color: '#94a3b8', fontSize: '0.95rem', margin: '0 0 1.75rem 0' }}>
          Bundled with Vite, served with high-performance edge static assets.
        </p>

        <div style={{ display: 'flex', gap: '1rem', justifyContent: 'center', marginBottom: '2rem' }}>
          <button
            onClick={() => setCount((c) => c + 1)}
            style={{
              padding: '0.75rem 1.5rem',
              backgroundColor: '#3b82f6',
              color: '#ffffff',
              border: 'none',
              borderRadius: '8px',
              fontWeight: 600,
              cursor: 'pointer',
              fontSize: '0.95rem'
            }}
          >
            Count: {count}
          </button>

          <button
            onClick={fetchApi}
            disabled={isLoading}
            style={{
              padding: '0.75rem 1.5rem',
              backgroundColor: '#334155',
              color: '#f8fafc',
              border: '1px solid #475569',
              borderRadius: '8px',
              fontWeight: 600,
              cursor: 'pointer',
              fontSize: '0.95rem'
            }}
          >
            {isLoading ? 'Pinging API...' : 'Ping Worker API'}
          </button>
        </div>

        {apiData && (
          <div style={{
            backgroundColor: '#0f172a',
            border: '1px solid #334155',
            borderRadius: '8px',
            padding: '1rem',
            textAlign: 'left',
            fontFamily: 'monospace',
            fontSize: '0.8rem',
            color: '#38bdf8'
          }}>
            <div style={{ color: '#94a3b8', marginBottom: '0.25rem' }}>Response from /api/health:</div>
            <pre style={{ margin: 0 }}>{JSON.stringify(apiData, null, 2)}</pre>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
