'use strict';

const fs = require('node:fs');
const path = require('node:path');

const [workspace, parentPid, siyuanPort] = process.argv.slice(2);
const confPath = path.join(workspace, 'conf', 'conf.json');
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function main() {
    if (!workspace || !parentPid || !siyuanPort) throw new Error('missing launcher arguments');
    let token = '';
    let ready = false;
    for (let attempt = 0; attempt < 60; attempt += 1) {
        try {
            token = JSON.parse(fs.readFileSync(confPath, 'utf8')).api?.token ?? '';
            if (token) {
                const response = await fetch(`http://127.0.0.1:${siyuanPort}/api/system/version`, {
                    method: 'POST',
                    headers: { Authorization: `Token ${token}`, 'Content-Type': 'application/json' },
                    body: '{}',
                });
                await response.body?.cancel();
                if (response.ok) {
                    ready = true;
                    break;
                }
            }
        } catch {
            token = '';
        }
        await wait(1000);
    }
    if (!ready) throw new Error('SiYuan API did not become ready');

    Object.assign(process.env, {
        SIYUAN_API_URL: `http://127.0.0.1:${siyuanPort}`,
        SIYUAN_TOKEN: token,
        SIYUAN_MCP_TRANSPORT: 'http',
        SIYUAN_MCP_HOST: '127.0.0.1',
        SIYUAN_MCP_PORT: '36806',
        SIYUAN_MCP_PATH: '/mcp',
        SIYUAN_MCP_PARENT_PID: parentPid,
    });
    await require('./sisyphus/mcp-server.cjs').startMcpServer();
}

main().catch((error) => {
    console.error('[Sisyphus]', error instanceof Error ? error.message : String(error));
    process.exit(1);
});
