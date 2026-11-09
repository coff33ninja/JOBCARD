import { ApiClient } from './client.ts';
import { MockApi } from './mockApi.ts';
import { HttpApi } from './httpApi.ts';
import rawSeedData from '../../seed/seed.json';
import { RawSeedData } from './seedLoader.ts';

let clientInstance: ApiClient | null = null;

export function createApiClient(mode?: 'mock' | 'http', apiUrl?: string): ApiClient {
  const selectedMode = mode || (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API) || 'mock';
  const selectedUrl = apiUrl || (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API_URL) || '';

  if (selectedMode === 'http') {
    const selectedToken = (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API_TOKEN) || '';
    return new HttpApi(selectedUrl, selectedToken);
  }

  return new MockApi(rawSeedData as unknown as RawSeedData);
}

export function getApiClient(): ApiClient {
  if (!clientInstance) {
    clientInstance = createApiClient();
  }
  return clientInstance;
}

export function resetApiClient(): void {
  clientInstance = null;
}
