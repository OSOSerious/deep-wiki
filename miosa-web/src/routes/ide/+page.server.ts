import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
  // IDE configuration can be loaded here if needed
  return {
    ideServerUrl: process.env.IDE_SERVER_URL || 'http://localhost:8085',
    apiEndpoint: '/api/ide'
  };
};