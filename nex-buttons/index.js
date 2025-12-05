const {
    default: makeWASocket,
    useMultiFileAuthState,
    DisconnectReason,
    jidNormalizedUser,
    delay,
    makeCacheableSignalKeyStore,
    fetchLatestBaileysVersion
} = require('@whiskeysockets/baileys');
const pino = require('pino');
const express = require('express');
const fs = require('fs');
const bodyParser = require('body-parser');
const cors = require('cors');

// 🔥 Importa o helper de botões que FUNCIONA
const { sendButtons, sendInteractiveMessage } = require('@ryuu-reinzz/button-helper');

const app = express();
app.use(cors());
app.use(bodyParser.json({ limit: '50mb' }));

const PORT = 3001;

// --- ESTRUTURAS DE DADOS ---
const sessions = new Map();
const qrCodes = new Map();
const retryCounters = new Map();

// Store Manual (RAM)
const localStore = new Map();

function getStore(instanceId) {
    if (!localStore.has(instanceId)) {
        localStore.set(instanceId, { contacts: {}, messages: {}, chats: {} });
    }
    return localStore.get(instanceId);
}

// ============================================
// FUNÇÃO DE CONEXÃO
// ============================================

async function startSession(instanceId) {
    // Evita duplicidade se já estiver conectado e saudável
    if (sessions.has(instanceId) && !sessions.get(instanceId).ws?.isClosed) {
        return sessions.get(instanceId);
    }

    const authPath = `auth_info/${instanceId}`;
    if (!fs.existsSync(authPath)) {
        fs.mkdirSync(authPath, { recursive: true });
    }

    const { state, saveCreds } = await useMultiFileAuthState(authPath);
    const { version } = await fetchLatestBaileysVersion();

    console.log(`[${instanceId}] Iniciando Engine (v${version.join('.')})`);

    const sock = makeWASocket({
        version,
        logger: pino({ level: 'silent' }),
        auth: {
            creds: state.creds,
            keys: makeCacheableSignalKeyStore(state.keys, pino({ level: "silent" })),
        },
        browser: ["NexusWa_Api", "Chrome", "1.0.0"],
        markOnlineOnConnect: true,
        generateHighQualityLinkPreview: true,
        syncFullHistory: true, 
        connectTimeoutMs: 60000,
        defaultQueryTimeoutMs: 60000,
        retryRequestDelayMs: 500,
        getMessage: async (key) => {
            const storeData = getStore(instanceId);
            if (storeData.messages[key.id]) {
                return storeData.messages[key.id].message;
            }
            return { conversation: 'hello' };
        }
    });

    // --- EVENTOS ---

    sock.ev.on('creds.update', saveCreds);

    // 🔥 Carga Inicial de Contatos
    sock.ev.on('contacts.set', ({ contacts }) => {
        const storeData = getStore(instanceId);
        for (const contact of contacts) {
            storeData.contacts[contact.id] = { 
                ...(storeData.contacts[contact.id] || {}), 
                ...contact 
            };
        }
        console.log(`[${instanceId}] ✅ ${contacts.length} contatos sincronizados.`);
    });

    sock.ev.on('contacts.upsert', (contacts) => {
        const storeData = getStore(instanceId);
        for (const contact of contacts) {
            storeData.contacts[contact.id] = { 
                ...(storeData.contacts[contact.id] || {}), 
                ...contact 
            };
        }
        console.log(`[${instanceId}] 📇 ${contacts.length} contatos atualizados.`);
    });

    // 🔥 Captura mensagens novas em tempo real
    sock.ev.on('messages.upsert', ({ messages }) => {
        const storeData = getStore(instanceId);
        messages.forEach(m => {
            if (m.key && m.key.id) storeData.messages[m.key.id] = m;
        });
        console.log(`[${instanceId}] 📩 Mensagens novas: ${messages.length}`);
    });

    // 🔥 Atualização automática de mensagens
    sock.ev.on('messages.update', updates => {
        const storeData = getStore(instanceId);
        updates.forEach(u => {
            if (u.key?.id && storeData.messages[u.key.id]) {
                storeData.messages[u.key.id] = { ...storeData.messages[u.key.id], ...u };
            }
        });
    });

    // 🔥 Sincronização de Chats
    sock.ev.on('chats.set', ({ chats }) => {
        const storeData = getStore(instanceId);
        for (const chat of chats) {
            storeData.chats[chat.id] = {
                ...(storeData.chats[chat.id] || {}),
                ...chat
            };
        }
        console.log(`[${instanceId}] 💬 ${chats.length} chats sincronizados.`);
    });

    // 🔥 Captura chats incrementais
    sock.ev.on('chats.upsert', (chats) => {
        const storeData = getStore(instanceId);
        chats.forEach(chat => {
            storeData.chats[chat.id] = chat;
        });
        console.log(`[${instanceId}] 🗂 Chats atualizados: ${Object.keys(storeData.chats).length}`);
    });

    // Controle de Conexão
    sock.ev.on('connection.update', (update) => {
        const { connection, lastDisconnect, qr } = update;

        if (qr) {
            console.log(`[${instanceId}] 📷 QR Code gerado!`);
            qrCodes.set(instanceId, qr);
            retryCounters.set(instanceId, 0);
        }
        
        if (connection === 'connecting') {
            console.log(`[${instanceId}] ⏳ Conectando...`);
        }

        if (connection === 'open') {
            console.log(`[${instanceId}] 🟢 CONECTADO COMPLETAMENTE`);
            qrCodes.delete(instanceId);
            retryCounters.set(instanceId, 0);

            const storeData = getStore(instanceId);

            // Espera 3s para store ser carregada e busca grupos
            setTimeout(async () => {
                try {
                    const groups = await sock.groupFetchAllParticipating();
                    Object.values(groups).forEach(g => storeData.chats[g.id] = g);
                    console.log(`[${instanceId}] 🔥 Grupos carregados: ${Object.keys(groups).length}`);
                } catch(e) { 
                    console.log("Erro grupos:", e.message); 
                }
            }, 3000);
        }

        if (connection === 'close') {
            const reason = lastDisconnect?.error?.message || "";
            const statusCode = lastDisconnect?.error?.output?.statusCode;
            
            const isLoggedOut = reason.includes("logged out") || 
                                reason.includes("401") || 
                                reason.includes("403") || 
                                statusCode === DisconnectReason.loggedOut;

            const shouldReconnect = !isLoggedOut;

            console.log(`[${instanceId}] 🔴 DESCONECTADO | Motivo: ${reason || statusCode} | Reconnect: ${shouldReconnect}`);

            // Limpeza de memória
            sessions.delete(instanceId);
            qrCodes.delete(instanceId);

            if (shouldReconnect) {
                const retries = retryCounters.get(instanceId) || 0;
                if (retries < 10) {
                    retryCounters.set(instanceId, retries + 1);
                    console.log(`[${instanceId}] Tentando reconectar em 2s (${retries+1}/10)...`);
                    setTimeout(() => startSession(instanceId), 2000);
                } else {
                    console.log(`[${instanceId}] ⛔ Limite de tentativas excedido.`);
                }
            } else {
                console.log(`[${instanceId}] Sessão encerrada (Logout). Limpando arquivos.`);
                try { fs.rmSync(authPath, { recursive: true, force: true }); } catch(e) {}
            }
        }
    });

    sessions.set(instanceId, sock);
    return sock;
}

// ============================================
// ROTAS API
// ============================================

// --- SESSÃO ---

app.post('/session/start', async (req, res) => {
    const { instance } = req.body;
    if (!instance) return res.status(400).json({ error: 'Instance required' });

    console.log(`>>> [API] Pedido de conexão: ${instance}`);
    retryCounters.set(instance, 0);

    try {
        const sock = await startSession(instance);
        
        if (sock.authState.creds.me) {
            return res.json({ status: 'CONNECTED', qrcode: '' });
        }

        let attempts = 0;
        while (attempts < 30) {
            const qr = qrCodes.get(instance);
            if (qr) return res.json({ status: 'QRCODE', qrcode: qr });
            if (sock.authState.creds.me) return res.json({ status: 'CONNECTED', qrcode: '' });
            await delay(500);
            attempts++;
        }
        
        res.json({ status: 'TIMEOUT', qrcode: '' });

    } catch (error) {
        console.error(`[${instance}] Erro fatal:`, error);
        res.status(500).json({ error: error.message });
    }
});

app.post('/session/pair-code', async (req, res) => {
    const { instance, phoneNumber } = req.body;
    if (!instance || !phoneNumber) return res.status(400).json({ error: 'Dados faltantes' });
    retryCounters.set(instance, 0);

    try {
        const sock = await startSession(instance);
        if (sock.authState.creds.me) return res.status(400).json({ error: 'Já conectado' });
        
        let attempts = 0;
        while (!sock.ws.isOpen && attempts < 10) { await delay(500); attempts++; }

        if (sock.ws.isOpen) {
            const code = await sock.requestPairingCode(phoneNumber);
            res.json({ status: 'success', code: code });
        } else {
            res.status(500).json({ error: 'Socket não abriu' });
        }
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

app.post('/session/logout', async (req, res) => {
    const { instance } = req.body;
    const sock = sessions.get(instance);
    if (sock) {
        try { 
            await sock.logout(); 
            sock.end();
        } catch(e) {}
        
        sessions.delete(instance);
        qrCodes.delete(instance);
        localStore.delete(instance);
        try { fs.rmSync(`auth_info/${instance}`, { recursive: true, force: true }); } catch(e) {}
        res.json({ status: 'success' });
    } else {
        res.json({ status: 'ignored' });
    }
});

// --- INFO ---

app.get('/v1/instance/:instance/info', async (req, res) => {
    const { instance } = req.params;
    const sock = sessions.get(instance);
    if (!sock || !sock.authState.creds.me) return res.status(404).json({ status: 'disconnected' });

    const jid = jidNormalizedUser(sock.authState.creds.me.id);
    let avatar = '';
    try { avatar = await sock.profilePictureUrl(jid, 'image'); } catch {}

    const storeData = getStore(instance);
    const contactsCount = Object.keys(storeData.contacts).length;
    const chatsCount = Object.keys(storeData.chats || {}).length;

    res.json({
        status: 'connected', 
        jid, 
        name: sock.authState.creds.me.name || instance,
        avatar, 
        contacts: contactsCount, 
        groups: chatsCount,
        sent: storeData.sentCount || 0
    });
});

// --- MENSAGENS ---

// Texto simples
app.post('/v1/message/text', async (req, res) => {
    const { instance, number, text } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });

    const storeData = getStore(instance);

    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;
        const sent = await sock.sendMessage(jid, { text });

        storeData.sentCount = (storeData.sentCount || 0) + 1;

        res.json({ 
            status: 'success',
            key: sent.key,
            totalSent: storeData.sentCount
        });

    } catch(e) { 
        res.status(500).json({ error: e.message }); 
    }
});

// 🔥 Botões usando @ryuu-reinzz/button-helper (FUNCIONA!)
app.post('/v1/message/buttons', async (req, res) => {
    const { instance, number, message, footer, buttons, title } = req.body;

    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Instância desconectada' });

    const jid = number.includes('@') ? number : number + '@s.whatsapp.net';

    try {
        // Usa o helper que funciona!
        const result = await sendButtons(sock, jid, {
            title: title || '',
            text: message,
            footer: footer || 'NexusWA',
            buttons: buttons.map(b => ({
                id: b.id,
                text: b.text
            }))
        });

        const storeData = getStore(instance);
        storeData.sentCount = (storeData.sentCount || 0) + 1;

        return res.json({ 
            status: 'success', 
            messageId: result?.key?.id || 'sent'
        });
        
    } catch (e) {
        console.error("Erro ao enviar botões:", e);
        return res.status(500).json({ error: e.message });
    }
});

// 🔥 Lista de Seleção
app.post('/v1/message/list', async (req, res) => {
    const { instance, number, title, message, footer, buttonText, sections } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });

    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;

        // Usa sendInteractiveMessage para listas
        const result = await sendInteractiveMessage(sock, jid, {
            text: message,
            footer: footer || 'NexusWA',
            title: title || '',
            interactiveButtons: [{
                name: 'single_select',
                buttonParamsJson: JSON.stringify({
                    title: buttonText || 'Selecionar',
                    sections: sections.map(section => ({
                        title: section.title || 'Opções',
                        rows: (section.rows || []).map(row => ({
                            title: row.title,
                            description: row.description || '',
                            id: row.id || row.rowId
                        }))
                    }))
                })
            }]
        });

        const storeData = getStore(instance);
        storeData.sentCount = (storeData.sentCount || 0) + 1;

        res.json({ status: "success", messageId: result?.key?.id || 'sent' });
    } catch (e) {
        console.error("Erro ao enviar lista:", e);
        res.status(500).json({ error: e.message });
    }
});

// 🔥 Botão com URL
app.post('/v1/message/url-button', async (req, res) => {
    const { instance, number, message, footer, title, buttonText, url } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });

    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;

        const result = await sendInteractiveMessage(sock, jid, {
            text: message,
            footer: footer || 'NexusWA',
            title: title || '',
            interactiveButtons: [{
                name: 'cta_url',
                buttonParamsJson: JSON.stringify({
                    display_text: buttonText || 'Acessar',
                    url: url,
                    merchant_url: url
                })
            }]
        });

        res.json({ status: "success", messageId: result?.key?.id || 'sent' });
    } catch (e) {
        console.error("Erro ao enviar botão URL:", e);
        res.status(500).json({ error: e.message });
    }
});

// 🔥 Botão de Copiar
app.post('/v1/message/copy-button', async (req, res) => {
    const { instance, number, message, footer, title, buttonText, copyCode } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });

    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;

        const result = await sendInteractiveMessage(sock, jid, {
            text: message,
            footer: footer || 'NexusWA',
            title: title || '',
            interactiveButtons: [{
                name: 'cta_copy',
                buttonParamsJson: JSON.stringify({
                    display_text: buttonText || 'Copiar',
                    copy_code: copyCode
                })
            }]
        });

        res.json({ status: "success", messageId: result?.key?.id || 'sent' });
    } catch (e) {
        console.error("Erro ao enviar botão copiar:", e);
        res.status(500).json({ error: e.message });
    }
});

// Interactive (compatibilidade)
app.post('/v1/message/interactive', async (req, res) => {
    const { instance, number, interactive } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });
    
    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;
        
        // Se tiver botões no formato interactive, converte para o helper
        if (interactive.action?.buttons) {
            const buttons = interactive.action.buttons.map(b => ({
                id: b.id || b.buttonParamsJson,
                text: b.name || b.title || b.text
            }));
            
            const result = await sendButtons(sock, jid, {
                text: interactive.body?.text || interactive.text || '',
                footer: interactive.footer?.text || interactive.footer || 'NexusWA',
                title: interactive.header?.text || '',
                buttons: buttons
            });
            
            return res.json({ status: 'success', messageId: result?.key?.id || 'sent' });
        }
        
        // Fallback para texto simples
        const sent = await sock.sendMessage(jid, { 
            text: interactive.body?.text || interactive.text || 'Mensagem interativa'
        });
        res.json({ status: 'success', key: sent.key });
        
    } catch(e) { 
        console.error("Erro interactive:", e);
        res.status(500).json({ error: e.message }); 
    }
});

// --- CONTATOS E GRUPOS ---

app.get('/v1/contacts/:instance', async (req, res) => {
    const { instance } = req.params;
    const storeData = getStore(instance);
    const sock = sessions.get(instance);
    
    // Contatos individuais
    const contacts = Object.values(storeData.contacts).map(c => ({
        jid: c.id,
        name: c.name || c.notify || c.verifiedName || c.id.split('@')[0],
        is_group: c.id.endsWith('@g.us')
    }));
    
    // Adiciona grupos do storeData.chats
    const groupsFromChats = Object.values(storeData.chats || {})
        .filter(chat => chat.id && chat.id.endsWith('@g.us'))
        .map(g => ({
            jid: g.id,
            name: g.subject || g.name || g.id.split('@')[0],
            is_group: true
        }));
    
    // Se não tiver grupos no cache, busca do socket diretamente
    let groupsFromSocket = [];
    if (groupsFromChats.length === 0 && sock) {
        try {
            const groups = await sock.groupFetchAllParticipating();
            groupsFromSocket = Object.values(groups).map(g => ({
                jid: g.id,
                name: g.subject || g.id.split('@')[0],
                is_group: true
            }));
            
            // Salva no cache
            Object.values(groups).forEach(g => storeData.chats[g.id] = g);
        } catch(e) {
            console.log(`[${instance}] Erro ao buscar grupos:`, e.message);
        }
    }
    
    // Combina contatos + grupos (evita duplicatas)
    const allGroups = [...groupsFromChats, ...groupsFromSocket];
    const groupJids = new Set(allGroups.map(g => g.jid));
    const filteredContacts = contacts.filter(c => !groupJids.has(c.jid));
    const result = [...filteredContacts, ...allGroups];
    
    res.json(result);
});

app.get('/v1/groups/:instance', async (req, res) => {
    const { instance } = req.params;
    const sock = sessions.get(instance);
    if (!sock) return res.json([]);
    try {
        const groups = await sock.groupFetchAllParticipating();
        const result = Object.values(groups).map(g => ({
            jid: g.id, 
            name: g.subject, 
            participants: g.participants.length,
            owner: g.owner, 
            created: g.creation
        }));
        res.json(result);
    } catch(e) { res.json([]); }
});

// --- MENSAGENS ---

app.get('/v1/messages/:instance/:jid', (req, res) => {
    const { instance, jid } = req.params;
    const storeData = getStore(instance);

    const msgs = Object.values(storeData.messages)
        .filter(m => m.key?.remoteJid === jid)
        .sort((a, b) => (a.messageTimestamp || 0) - (b.messageTimestamp || 0));

    res.json(msgs);
});

// ============================================
// AUTO-RECOVERY
// ============================================

if (!fs.existsSync('auth_info')) fs.mkdirSync('auth_info');
const folders = fs.readdirSync('auth_info');
folders.forEach(f => {
    if (fs.lstatSync(`auth_info/${f}`).isDirectory()) {
        console.log(`Recuperando sessão: ${f}`);
        startSession(f);
    }
});

app.listen(PORT, () => console.log(`🚀 Nexus Baileys API rodando na porta ${PORT}`));