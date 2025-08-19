import * as server from '../entries/pages/build/schema/_page.server.ts.js';

export const index = 7;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/build/schema/_page.svelte.js')).default;
export { server };
export const server_id = "src/routes/build/schema/+page.server.ts";
export const imports = ["_app/immutable/nodes/7.CAG5_cGP.js","_app/immutable/chunks/Bzak7iHL.js","_app/immutable/chunks/69_IOA4Y.js","_app/immutable/chunks/DIeogL5L.js"];
export const stylesheets = [];
export const fonts = [];
