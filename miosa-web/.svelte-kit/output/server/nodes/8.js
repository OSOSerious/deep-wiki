import * as server from '../entries/pages/chat/_page.server.ts.js';

export const index = 8;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/chat/_page.svelte.js')).default;
export { server };
export const server_id = "src/routes/chat/+page.server.ts";
export const imports = ["_app/immutable/nodes/8.CAG5_cGP.js","_app/immutable/chunks/Bzak7iHL.js","_app/immutable/chunks/69_IOA4Y.js","_app/immutable/chunks/DIeogL5L.js"];
export const stylesheets = [];
export const fonts = [];
