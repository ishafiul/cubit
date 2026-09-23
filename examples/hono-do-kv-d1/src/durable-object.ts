/**
 * Durable Object implementation: Stateful Counter with persistent storage
 */

export class CounterDO {
  state: DurableObjectState;
  count: number = 0;

  constructor(state: DurableObjectState, env: any) {
    this.state = state;
    this.state.blockConcurrencyWhile(async () => {
      const stored = await this.state.storage.get<number>('count');
      this.count = stored || 0;
    });
  }

  // Handle incoming HTTP fetch from Worker
  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);

    if (url.pathname.endsWith('/increment') || request.method === 'POST') {
      this.count++;
      await this.state.storage.put('count', this.count);
      return Response.json({
        action: 'incremented',
        count: this.count,
        timestamp: new Date().toISOString(),
      });
    }

    if (url.pathname.endsWith('/reset')) {
      this.count = 0;
      await this.state.storage.delete('count');
      return Response.json({
        action: 'reset',
        count: this.count,
      });
    }

    // Default: return current state
    return Response.json({
      count: this.count,
      id: this.state.id.toString(),
    });
  }
}
