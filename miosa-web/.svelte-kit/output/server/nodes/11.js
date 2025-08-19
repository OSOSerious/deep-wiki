import * as universal from '../entries/pages/ide/_page.ts.js';
import * as server from '../entries/pages/ide/_page.server.ts.js';

export const index = 11;
let component_cache;
export const component = async () => component_cache ??= (await import('../entries/pages/ide/_page.svelte.js')).default;
export { universal };
export const universal_id = "src/routes/ide/+page.ts";
export { server };
export const server_id = "src/routes/ide/+page.server.ts";
export const imports = ["_app/immutable/nodes/11.Cb-T5f9w.js","_app/immutable/chunks/Bzak7iHL.js","_app/immutable/chunks/69_IOA4Y.js","_app/immutable/chunks/DIeogL5L.js"];
export const stylesheets = [];
export const fonts = [];
