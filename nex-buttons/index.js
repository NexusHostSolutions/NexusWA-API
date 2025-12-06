const {
    default: makeWASocket,
    useMultiFileAuthState,
    DisconnectReason,
    jidNormalizedUser,
    delay,
    makeCacheableSignalKeyStore,
    fetchLatestBaileysVersion
} = require('@whiskeysockets/baileys');
require('dotenv').config();
const pino = require('pino');
const express = require('express');
const fs = require('fs');
const bodyParser = require('body-parser');
const cors = require('cors');
const crypto = require('crypto');

const { sendButtons, sendInteractiveMessage } = require('@ryuu-reinzz/button-helper');

// ============================================
// POSTGRESQL - Pool de Conexão
// ============================================
const { Pool } = require('pg');

const pool = new Pool({
    host: process.env.DB_HOST,
    port: process.env.DB_PORT,
    database: process.env.DB_NAME,
    user: process.env.DB_USER,
    password: process.env.DB_PASS,
    max: 20,
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 2000,
});

pool.query('SELECT NOW()')
    .then(() => console.log('✅ PostgreSQL conectado!'))
    .catch(err => console.error('❌ Erro PostgreSQL:', err.message));

// ============================================
// FUNÇÕES DE HASH E GERAÇÃO DE KEYS
// ============================================

function hashKey(key) {
    return crypto.createHash('sha256').update(key).digest('hex');
}

function generateApiKey(tipo = 'user') {
    const prefix = tipo === 'super_admin' ? 'nxsa_' : 'nxus_';
    const randomPart = crypto.randomBytes(24).toString('hex');
    return `${prefix}${randomPart}`;
}

function getKeyPrefix(key) {
    return key.substring(0, 8);
}

// ============================================
// FUNÇÕES DE BANCO - API KEYS
// ============================================

async function createApiKey(nome, tipo = 'user', validadeDias = null, instanciasPermitidas = null) {
    try {
        const key = generateApiKey(tipo);
        const keyHash = hashKey(key);
        const keyPrefix = getKeyPrefix(key);
        
        let expiraEm = null;
        if (validadeDias) {
            expiraEm = new Date();
            expiraEm.setDate(expiraEm.getDate() + validadeDias);
        }

        const result = await pool.query(`
            INSERT INTO api_keys (key_hash, key_prefix, nome, tipo, instancias_permitidas, expira_em)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id, key_prefix, nome, tipo, ativa, instancias_permitidas, criado_em, expira_em
        `, [keyHash, keyPrefix, nome, tipo, instanciasPermitidas, expiraEm]);

        return {
            ...result.rows[0],
            key: key // Retorna a key em texto apenas na criação!
        };
    } catch (err) {
        console.error('[DB] Erro ao criar API Key:', err.message);
        return null;
    }
}

async function validateApiKey(key) {
    try {
        const keyHash = hashKey(key);
        const result = await pool.query(`
            SELECT id, key_prefix, nome, tipo, ativa, instancias_permitidas, criado_em, expira_em, ultimo_uso
            FROM api_keys 
            WHERE key_hash = $1
        `, [keyHash]);

        if (result.rows.length === 0) {
            return { valid: false, error: 'API Key inválida' };
        }

        const apiKey = result.rows[0];

        // Verifica se está ativa
        if (!apiKey.ativa) {
            return { valid: false, error: 'API Key desativada' };
        }

        // Verifica expiração
        if (apiKey.expira_em && new Date(apiKey.expira_em) < new Date()) {
            return { valid: false, error: 'API Key expirada' };
        }

        // Atualiza último uso e contador
        await pool.query(`
            UPDATE api_keys 
            SET ultimo_uso = NOW(), total_requisicoes = total_requisicoes + 1 
            WHERE id = $1
        `, [apiKey.id]);

        return { 
            valid: true, 
            apiKey: apiKey,
            isSuperAdmin: apiKey.tipo === 'super_admin'
        };
    } catch (err) {
        console.error('[DB] Erro ao validar API Key:', err.message);
        return { valid: false, error: 'Erro interno' };
    }
}

async function getApiKeyById(id) {
    try {
        const result = await pool.query(`
            SELECT id, key_prefix, nome, tipo, ativa, instancias_permitidas, criado_em, expira_em, ultimo_uso, total_requisicoes
            FROM api_keys WHERE id = $1
        `, [id]);
        return result.rows[0];
    } catch (err) {
        return null;
    }
}

async function getAllApiKeys() {
    try {
        const result = await pool.query(`
            SELECT id, key_prefix, nome, tipo, ativa, instancias_permitidas, criado_em, expira_em, ultimo_uso, total_requisicoes
            FROM api_keys 
            ORDER BY criado_em DESC
        `);
        return result.rows;
    } catch (err) {
        return [];
    }
}

async function updateApiKey(id, data) {
    try {
        const fields = [];
        const values = [];
        let idx = 1;

        if (data.nome !== undefined) { fields.push(`nome = $${idx++}`); values.push(data.nome); }
        if (data.ativa !== undefined) { fields.push(`ativa = $${idx++}`); values.push(data.ativa); }
        if (data.tipo !== undefined) { fields.push(`tipo = $${idx++}`); values.push(data.tipo); }
        if (data.instancias_permitidas !== undefined) { fields.push(`instancias_permitidas = $${idx++}`); values.push(data.instancias_permitidas); }
        if (data.expira_em !== undefined) { fields.push(`expira_em = $${idx++}`); values.push(data.expira_em); }

        if (fields.length === 0) return null;

        values.push(id);
        const query = `UPDATE api_keys SET ${fields.join(', ')} WHERE id = $${idx} RETURNING *`;
        const result = await pool.query(query, values);
        return result.rows[0];
    } catch (err) {
        return null;
    }
}

async function deleteApiKey(id) {
    try {
        await pool.query(`DELETE FROM api_keys WHERE id = $1`, [id]);
        return true;
    } catch (err) {
        return false;
    }
}

async function ensureSuperAdminExists() {
    try {
        const result = await pool.query(`SELECT id FROM api_keys WHERE tipo = 'super_admin' LIMIT 1`);
        if (result.rows.length === 0) {
            console.log('⚠️ Nenhum Super Admin encontrado. Criando...');
            const admin = await createApiKey('Super Admin Padrão', 'super_admin', null, null);
            if (admin) {
                console.log('');
                console.log('═══════════════════════════════════════════════════════════');
                console.log('🔐 SUPER ADMIN API KEY CRIADA - GUARDE COM SEGURANÇA!');
                console.log('═══════════════════════════════════════════════════════════');
                console.log(`📋 Key: ${admin.key}`);
                console.log('═══════════════════════════════════════════════════════════');
                console.log('⚠️  Esta key só será mostrada UMA VEZ!');
                console.log('═══════════════════════════════════════════════════════════');
                console.log('');
            }
        }
    } catch (err) {
        console.error('Erro ao verificar Super Admin:', err.message);
    }
}

// ============================================
// FUNÇÕES DE BANCO - INSTÂNCIAS
// ============================================

async function createInstance(nome, criadoPor = null) {
    try {
        const result = await pool.query(`
            INSERT INTO instancias (nome, criado_por)
            VALUES ($1, $2)
            ON CONFLICT (nome) DO UPDATE SET atualizado_em = NOW()
            RETURNING *
        `, [nome, criadoPor]);
        return result.rows[0];
    } catch (err) {
        console.error('[DB] Erro ao criar instância:', err.message);
        return null;
    }
}

async function updateInstance(nome, data) {
    try {
        const fields = [];
        const values = [];
        let idx = 1;

        if (data.status !== undefined) { fields.push(`status = $${idx++}`); values.push(data.status); }
        if (data.push_name !== undefined) { fields.push(`push_name = $${idx++}`); values.push(data.push_name); }
        if (data.jid !== undefined) { fields.push(`jid = $${idx++}`); values.push(data.jid); }
        if (data.avatar !== undefined) { fields.push(`avatar = $${idx++}`); values.push(data.avatar); }
        if (data.webhook_url !== undefined) { fields.push(`webhook_url = $${idx++}`); values.push(data.webhook_url); }
        if (data.webhook_enabled !== undefined) { fields.push(`webhook_enabled = $${idx++}`); values.push(data.webhook_enabled); }

        fields.push(`atualizado_em = NOW()`);
        values.push(nome);

        const query = `UPDATE instancias SET ${fields.join(', ')} WHERE nome = $${idx} RETURNING *`;
        const result = await pool.query(query, values);
        return result.rows[0];
    } catch (err) {
        return null;
    }
}

async function getInstance(nome) {
    try {
        const result = await pool.query(`SELECT * FROM instancias WHERE nome = $1`, [nome]);
        return result.rows[0];
    } catch (err) {
        return null;
    }
}

async function getAllInstances() {
    try {
        const result = await pool.query(`SELECT * FROM instancias ORDER BY criado_em DESC`);
        return result.rows;
    } catch (err) {
        return [];
    }
}

async function getInstancesForUser(instanciasPermitidas) {
    try {
        if (!instanciasPermitidas || instanciasPermitidas.length === 0) {
            return [];
        }
        const result = await pool.query(`
            SELECT * FROM instancias 
            WHERE nome = ANY($1)
            ORDER BY criado_em DESC
        `, [instanciasPermitidas]);
        return result.rows;
    } catch (err) {
        return [];
    }
}

async function deleteInstance(nome) {
    try {
        await pool.query(`DELETE FROM instancias WHERE nome = $1`, [nome]);
        await pool.query(`DELETE FROM contatos WHERE instance = $1`, [nome]);
        await pool.query(`DELETE FROM grupos WHERE instance = $1`, [nome]);
        await pool.query(`DELETE FROM mensagens WHERE instance = $1`, [nome]);
        return true;
    } catch (err) {
        return false;
    }
}

// ============================================
// FUNÇÕES DE BANCO - CONTATOS/GRUPOS/MENSAGENS
// ============================================

async function saveContact(instance, jid, nome) {
    try {
        await pool.query(`
            INSERT INTO contatos (instance, jid, nome)
            VALUES ($1, $2, $3)
            ON CONFLICT (instance, jid) 
            DO UPDATE SET nome = EXCLUDED.nome
        `, [instance, jid, nome || jid.split('@')[0]]);
    } catch (err) {}
}

async function saveGroup(instance, jid, nome) {
    try {
        await pool.query(`
            INSERT INTO grupos (instance, jid, nome)
            VALUES ($1, $2, $3)
            ON CONFLICT (instance, jid) 
            DO UPDATE SET nome = EXCLUDED.nome
        `, [instance, jid, nome || jid.split('@')[0]]);
    } catch (err) {}
}

async function saveMessage(instance, jid, tipo, conteudo) {
    try {
        await pool.query(`
            INSERT INTO mensagens (instance, jid, tipo, conteudo)
            VALUES ($1, $2, $3, $4)
        `, [instance, jid, tipo, conteudo]);
    } catch (err) {}
}

async function getContactsFromDB(instance) {
    try {
        const result = await pool.query(`
            SELECT jid, nome, criado_em FROM contatos WHERE instance = $1 ORDER BY nome ASC
        `, [instance]);
        return result.rows;
    } catch (err) {
        return [];
    }
}

async function getGroupsFromDB(instance) {
    try {
        const result = await pool.query(`
            SELECT jid, nome, criado_em FROM grupos WHERE instance = $1 ORDER BY nome ASC
        `, [instance]);
        return result.rows;
    } catch (err) {
        return [];
    }
}

async function getMessagesFromDB(instance, jid) {
    try {
        const result = await pool.query(`
            SELECT id, tipo, conteudo, criado_em FROM mensagens 
            WHERE instance = $1 AND jid = $2 ORDER BY criado_em DESC LIMIT 100
        `, [instance, jid]);
        return result.rows;
    } catch (err) {
        return [];
    }
}

async function getDBStats(instance) {
    try {
        const contactsResult = await pool.query(`SELECT COUNT(*) as count FROM contatos WHERE instance = $1`, [instance]);
        const groupsResult = await pool.query(`SELECT COUNT(*) as count FROM grupos WHERE instance = $1`, [instance]);
        const messagesResult = await pool.query(`SELECT COUNT(*) as count FROM mensagens WHERE instance = $1`, [instance]);
        
        return {
            contacts: parseInt(contactsResult.rows[0]?.count || 0),
            groups: parseInt(groupsResult.rows[0]?.count || 0),
            messages: parseInt(messagesResult.rows[0]?.count || 0)
        };
    } catch (err) {
        return { contacts: 0, groups: 0, messages: 0 };
    }
}

async function logAccess(apiKeyId, endpoint, metodo, ip, statusCode) {
    try {
        await pool.query(`
            INSERT INTO logs_acesso (api_key_id, endpoint, metodo, ip, status_code)
            VALUES ($1, $2, $3, $4, $5)
        `, [apiKeyId, endpoint, metodo, ip, statusCode]);
    } catch (err) {}
}

// ============================================
// EXPRESS APP
// ============================================

const app = express();
app.use(cors());
app.use(bodyParser.json({ limit: '50mb' }));

const PORT = 3001;

const sessions = new Map();
const qrCodes = new Map();
const retryCounters = new Map();
const localStore = new Map();
const syncStatus = new Map();

function getStore(instanceId) {
    if (!localStore.has(instanceId)) {
        localStore.set(instanceId, { contacts: {}, messages: {}, chats: {} });
    }
    return localStore.get(instanceId);
}

function getSyncStatus(instanceId) {
    if (!syncStatus.has(instanceId)) {
        syncStatus.set(instanceId, {
            syncing: false, progress: 0, total: 0, phase: '',
            completed: false, error: null, contactsSynced: 0, groupsSynced: 0
        });
    }
    return syncStatus.get(instanceId);
}

// ============================================
// MIDDLEWARE DE AUTENTICAÇÃO
// ============================================

const authMiddleware = async (req, res, next) => {
    const apiKey = req.headers['apikey'] || req.headers['x-api-key'] || req.query.apikey;
    
    if (!apiKey) {
        return res.status(401).json({ error: 'API Key não fornecida' });
    }

    const validation = await validateApiKey(apiKey);
    
    if (!validation.valid) {
        await logAccess(null, req.path, req.method, req.ip, 401);
        return res.status(401).json({ error: validation.error });
    }

    req.auth = validation;
    req.apiKeyData = validation.apiKey;
    req.isSuperAdmin = validation.isSuperAdmin;

    await logAccess(validation.apiKey.id, req.path, req.method, req.ip, 200);
    next();
};

// Middleware para verificar acesso à instância
const instanceAccessMiddleware = async (req, res, next) => {
    const instanceName = req.params.instance || req.body.instance;
    
    if (!instanceName) {
        return next();
    }

    // Super Admin tem acesso a tudo
    if (req.isSuperAdmin) {
        return next();
    }

    // Usuário normal: verifica se tem permissão
    const permitidas = req.apiKeyData.instancias_permitidas || [];
    if (!permitidas.includes(instanceName)) {
        return res.status(403).json({ error: 'Sem permissão para esta instância' });
    }

    next();
};

// Middleware apenas para Super Admin
const superAdminOnly = (req, res, next) => {
    if (!req.isSuperAdmin) {
        return res.status(403).json({ error: 'Acesso restrito a Super Admin' });
    }
    next();
};

// ============================================
// FUNÇÃO DE SINCRONIZAÇÃO COMPLETA
// ============================================

async function fullSync(sock, instanceId) {
    const status = getSyncStatus(instanceId);
    status.syncing = true;
    status.progress = 0;
    status.phase = 'Iniciando sincronização...';
    status.completed = false;
    status.error = null;
    status.contactsSynced = 0;
    status.groupsSynced = 0;

    console.log(`[${instanceId}] 🔄 Iniciando sincronização completa...`);

    try {
        // FASE 1: GRUPOS
        status.phase = 'Sincronizando grupos...';
        let groupCount = 0;
        try {
            const groups = await sock.groupFetchAllParticipating();
            const groupList = Object.values(groups);
            status.total = groupList.length;
            
            for (const group of groupList) {
                await saveGroup(instanceId, group.id, group.subject || group.id.split('@')[0]);
                groupCount++;
                status.progress = groupCount;
                status.groupsSynced = groupCount;
            }
            console.log(`[${instanceId}] ✅ ${groupCount} grupos sincronizados!`);
        } catch (e) {
            console.log(`[${instanceId}] ⚠️ Erro grupos:`, e.message);
        }

        // FASE 2: CONTATOS VIA CHATS
        status.phase = 'Sincronizando contatos...';
        status.progress = 0;
        let contactCount = 0;
        
        try {
            const storeData = getStore(instanceId);
            await delay(2000);
            
            const allChats = Object.values(storeData.chats || {});
            const allContacts = Object.values(storeData.contacts || {});
            const jidsToSync = new Set();
            
            for (const chat of allChats) {
                if (chat.id && !chat.id.endsWith('@g.us') && !chat.id.endsWith('@broadcast')) {
                    jidsToSync.add(chat.id);
                }
            }
            
            for (const contact of allContacts) {
                if (contact.id && !contact.id.endsWith('@g.us') && !contact.id.endsWith('@broadcast')) {
                    jidsToSync.add(contact.id);
                }
            }
            
            status.total = jidsToSync.size;
            
            for (const jid of jidsToSync) {
                const contact = storeData.contacts[jid] || {};
                const chat = storeData.chats[jid] || {};
                const nome = contact.name || contact.notify || contact.verifiedName || chat.name || jid.split('@')[0];
                
                await saveContact(instanceId, jid, nome);
                contactCount++;
                status.progress = contactCount;
                status.contactsSynced = contactCount;
            }
            
            console.log(`[${instanceId}] ✅ ${contactCount} contatos sincronizados!`);
        } catch (e) {
            console.log(`[${instanceId}] ⚠️ Erro contatos:`, e.message);
        }

        status.phase = 'Finalizando...';
        status.syncing = false;
        status.completed = true;
        
        const finalStats = await getDBStats(instanceId);
        console.log(`[${instanceId}] ✅ SINCRONIZAÇÃO COMPLETA! ${finalStats.contacts} contatos, ${finalStats.groups} grupos`);
        
        return { success: true, stats: finalStats };

    } catch (error) {
        status.syncing = false;
        status.error = error.message;
        return { success: false, error: error.message };
    }
}

// ============================================
// FUNÇÃO DE CONEXÃO
// ============================================

async function startSession(instanceId) {
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

    sock.ev.on('creds.update', saveCreds);

    // Contatos
    sock.ev.on('contacts.set', async ({ contacts }) => {
        const storeData = getStore(instanceId);
        let count = 0;
        for (const contact of contacts) {
            storeData.contacts[contact.id] = { ...(storeData.contacts[contact.id] || {}), ...contact };
            if (!contact.id.endsWith('@g.us') && !contact.id.endsWith('@broadcast')) {
                const nome = contact.name || contact.notify || contact.verifiedName || contact.id.split('@')[0];
                await saveContact(instanceId, contact.id, nome);
                count++;
            }
        }
        if (count > 0) console.log(`[${instanceId}] ✅ contacts.set: ${count} contatos`);
    });

    sock.ev.on('contacts.upsert', async (contacts) => {
        const storeData = getStore(instanceId);
        for (const contact of contacts) {
            storeData.contacts[contact.id] = { ...(storeData.contacts[contact.id] || {}), ...contact };
            if (!contact.id.endsWith('@g.us') && !contact.id.endsWith('@broadcast')) {
                const nome = contact.name || contact.notify || contact.verifiedName || contact.id.split('@')[0];
                await saveContact(instanceId, contact.id, nome);
            }
        }
    });

    // Mensagens
    sock.ev.on('messages.upsert', async ({ messages, type }) => {
        const storeData = getStore(instanceId);
        for (const m of messages) {
            if (m.key && m.key.id) {
                storeData.messages[m.key.id] = m;
                const jid = m.key.remoteJid;
                if (jid && jid !== 'status@broadcast') {
                    const isGroup = jid.endsWith('@g.us');
                    if (isGroup) {
                        const groupInfo = storeData.chats[jid];
                        await saveGroup(instanceId, jid, groupInfo?.subject || groupInfo?.name || jid.split('@')[0]);
                    } else {
                        const contactInfo = storeData.contacts[jid];
                        const nome = contactInfo?.name || contactInfo?.notify || contactInfo?.verifiedName || m.pushName || jid.split('@')[0];
                        await saveContact(instanceId, jid, nome);
                    }
                }
            }
        }
    });

    // Chats
    sock.ev.on('chats.set', async ({ chats }) => {
        const storeData = getStore(instanceId);
        for (const chat of chats) {
            storeData.chats[chat.id] = { ...(storeData.chats[chat.id] || {}), ...chat };
            if (chat.id.endsWith('@g.us')) {
                await saveGroup(instanceId, chat.id, chat.name || chat.subject || chat.id.split('@')[0]);
            } else if (!chat.id.endsWith('@broadcast')) {
                await saveContact(instanceId, chat.id, chat.name || chat.id.split('@')[0]);
            }
        }
    });

    sock.ev.on('chats.upsert', async (chats) => {
        const storeData = getStore(instanceId);
        for (const chat of chats) {
            storeData.chats[chat.id] = chat;
            if (chat.id.endsWith('@g.us')) {
                await saveGroup(instanceId, chat.id, chat.name || chat.subject || chat.id.split('@')[0]);
            } else if (!chat.id.endsWith('@broadcast')) {
                await saveContact(instanceId, chat.id, chat.name || chat.id.split('@')[0]);
            }
        }
    });

    // Conexão
    sock.ev.on('connection.update', async (update) => {
        const { connection, lastDisconnect, qr } = update;

        if (qr) {
            console.log(`[${instanceId}] 📷 QR Code gerado!`);
            qrCodes.set(instanceId, qr);
            retryCounters.set(instanceId, 0);
        }

        if (connection === 'connecting') {
            console.log(`[${instanceId}] ⏳ Conectando...`);
            await updateInstance(instanceId, { status: 'connecting' });
        }

        if (connection === 'open') {
            console.log(`[${instanceId}] 🟢 CONECTADO!`);
            qrCodes.delete(instanceId);
            retryCounters.set(instanceId, 0);

            const jid = jidNormalizedUser(sock.authState.creds.me.id);
            let avatar = '';
            try { avatar = await sock.profilePictureUrl(jid, 'image'); } catch {}

            await updateInstance(instanceId, {
                status: 'connected',
                jid: jid,
                push_name: sock.authState.creds.me.name || instanceId,
                avatar: avatar
            });

            setTimeout(async () => {
                await fullSync(sock, instanceId);
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

            console.log(`[${instanceId}] 🔴 DESCONECTADO | Motivo: ${reason || statusCode}`);

            await updateInstance(instanceId, { status: 'disconnected' });

            sessions.delete(instanceId);
            qrCodes.delete(instanceId);

            if (shouldReconnect) {
                const retries = retryCounters.get(instanceId) || 0;
                if (retries < 10) {
                    retryCounters.set(instanceId, retries + 1);
                    console.log(`[${instanceId}] Reconectando em 2s (${retries+1}/10)...`);
                    setTimeout(() => startSession(instanceId), 2000);
                }
            } else {
                console.log(`[${instanceId}] Sessão encerrada (Logout).`);
                try { fs.rmSync(authPath, { recursive: true, force: true }); } catch(e) {}
            }
        }
    });

    sessions.set(instanceId, sock);
    return sock;
}

// ============================================
// ROTAS PÚBLICAS (SEM AUTENTICAÇÃO)
// ============================================

app.get('/health', (req, res) => {
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// ============================================
// ROTAS DE API KEYS (SUPER ADMIN ONLY)
// ============================================

app.get('/v1/api-keys', authMiddleware, superAdminOnly, async (req, res) => {
    const keys = await getAllApiKeys();
    res.json(keys.map(k => ({
        id: k.id,
        prefix: k.key_prefix,
        nome: k.nome,
        tipo: k.tipo,
        ativa: k.ativa,
        instanciasPermitidas: k.instancias_permitidas,
        criadoEm: k.criado_em,
        expiraEm: k.expira_em,
        ultimoUso: k.ultimo_uso,
        totalRequisicoes: k.total_requisicoes,
        expirada: k.expira_em ? new Date(k.expira_em) < new Date() : false
    })));
});

app.post('/v1/api-keys', authMiddleware, superAdminOnly, async (req, res) => {
    const { nome, tipo = 'user', validade, instanciasPermitidas } = req.body;
    
    if (!nome) {
        return res.status(400).json({ error: 'Nome obrigatório' });
    }

    // Converte validade para dias
    let validadeDias = null;
    if (validade === '30') validadeDias = 30;
    else if (validade === '90') validadeDias = 90;
    else if (validade === '180') validadeDias = 180;
    else if (validade === '365') validadeDias = 365;
    // null = nunca expira

    const apiKey = await createApiKey(nome, tipo, validadeDias, instanciasPermitidas);
    
    if (apiKey) {
        res.json({
            status: 'success',
            message: 'API Key criada! Guarde a key, ela não será mostrada novamente.',
            apiKey: {
                id: apiKey.id,
                key: apiKey.key, // Só retorna a key na criação!
                prefix: apiKey.key_prefix,
                nome: apiKey.nome,
                tipo: apiKey.tipo,
                expiraEm: apiKey.expira_em
            }
        });
    } else {
        res.status(500).json({ error: 'Erro ao criar API Key' });
    }
});

app.put('/v1/api-keys/:id', authMiddleware, superAdminOnly, async (req, res) => {
    const { id } = req.params;
    const { nome, ativa, tipo, instanciasPermitidas, renovarValidade } = req.body;

    const updateData = {};
    if (nome !== undefined) updateData.nome = nome;
    if (ativa !== undefined) updateData.ativa = ativa;
    if (tipo !== undefined) updateData.tipo = tipo;
    if (instanciasPermitidas !== undefined) updateData.instancias_permitidas = instanciasPermitidas;
    
    // Renovar validade
    if (renovarValidade) {
        const novaExpiracao = new Date();
        novaExpiracao.setDate(novaExpiracao.getDate() + parseInt(renovarValidade));
        updateData.expira_em = novaExpiracao;
    }

    const updated = await updateApiKey(id, updateData);
    if (updated) {
        res.json({ status: 'success', apiKey: updated });
    } else {
        res.status(500).json({ error: 'Erro ao atualizar' });
    }
});

app.delete('/v1/api-keys/:id', authMiddleware, superAdminOnly, async (req, res) => {
    const { id } = req.params;
    
    // Não permite deletar a própria key
    if (parseInt(id) === req.apiKeyData.id) {
        return res.status(400).json({ error: 'Não é possível deletar sua própria API Key' });
    }

    const deleted = await deleteApiKey(id);
    if (deleted) {
        res.json({ status: 'success' });
    } else {
        res.status(500).json({ error: 'Erro ao deletar' });
    }
});

// ============================================
// ROTAS DE INSTÂNCIAS
// ============================================

app.get('/v1/instances', authMiddleware, async (req, res) => {
    let instances;
    
    if (req.isSuperAdmin) {
        instances = await getAllInstances();
    } else {
        instances = await getInstancesForUser(req.apiKeyData.instancias_permitidas);
    }
    
    const result = await Promise.all(instances.map(async (inst) => {
        const stats = await getDBStats(inst.nome);
        return {
            name: inst.nome,
            status: inst.status,
            jid: inst.jid,
            pushName: inst.push_name,
            avatar: inst.avatar,
            stats: {
                contacts: stats.contacts,
                groups: stats.groups,
                messagesSent: stats.messages
            },
            createdAt: inst.criado_em,
            updatedAt: inst.atualizado_em
        };
    }));
    
    res.json(result);
});

app.post('/v1/instances', authMiddleware, superAdminOnly, async (req, res) => {
    const { name } = req.body;
    if (!name) return res.status(400).json({ error: 'Nome obrigatório' });
    
    const instance = await createInstance(name, req.apiKeyData.id);
    
    if (instance) {
        res.json({ 
            status: 'success', 
            instance: {
                name: instance.nome,
                status: instance.status
            }
        });
    } else {
        res.status(500).json({ error: 'Erro ao criar instância' });
    }
});

app.delete('/v1/instances/:name', authMiddleware, superAdminOnly, async (req, res) => {
    const { name } = req.params;
    
    const sock = sessions.get(name);
    if (sock) {
        try { await sock.logout(); sock.end(); } catch(e) {}
        sessions.delete(name);
        qrCodes.delete(name);
        localStore.delete(name);
        syncStatus.delete(name);
    }
    
    try { fs.rmSync(`auth_info/${name}`, { recursive: true, force: true }); } catch(e) {}
    
    await deleteInstance(name);
    
    res.json({ status: 'success' });
});

// ============================================
// ROTAS DE SESSÃO
// ============================================

app.post('/session/start', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.body;
    if (!instance) return res.status(400).json({ error: 'Instance required' });

    console.log(`>>> [API] Pedido de conexão: ${instance}`);
    retryCounters.set(instance, 0);

    const existing = await getInstance(instance);
    if (!existing) {
        // Só Super Admin pode criar novas instâncias
        if (!req.isSuperAdmin) {
            return res.status(403).json({ error: 'Instância não existe. Contate o administrador.' });
        }
        await createInstance(instance, req.apiKeyData.id);
    }

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
        res.status(500).json({ error: error.message });
    }
});

app.post('/session/pair-code', authMiddleware, instanceAccessMiddleware, async (req, res) => {
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

app.post('/session/logout', authMiddleware, instanceAccessMiddleware, async (req, res) => {
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
        syncStatus.delete(instance);
        
        await updateInstance(instance, { status: 'disconnected', jid: null, avatar: null });
        
        try { fs.rmSync(`auth_info/${instance}`, { recursive: true, force: true }); } catch(e) {}
        res.json({ status: 'success' });
    } else {
        res.json({ status: 'ignored' });
    }
});

// ============================================
// ROTAS DE INFO E SYNC
// ============================================

app.get('/v1/instance/:instance/sync-status', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    const status = getSyncStatus(instance);
    const dbStats = await getDBStats(instance);
    res.json({ ...status, dbStats });
});

app.post('/v1/instance/:instance/sync', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Instância não conectada' });
    fullSync(sock, instance);
    res.json({ status: 'started' });
});

app.get('/v1/instance/:instance/info', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    const sock = sessions.get(instance);
    
    if (!sock || !sock.authState.creds.me) {
        const inst = await getInstance(instance);
        if (inst) {
            const stats = await getDBStats(instance);
            return res.json({
                status: 'disconnected',
                jid: inst.jid,
                name: inst.push_name,
                avatar: inst.avatar,
                contacts: stats.contacts,
                groups: stats.groups,
                messages: stats.messages
            });
        }
        return res.status(404).json({ status: 'disconnected' });
    }

    const jid = jidNormalizedUser(sock.authState.creds.me.id);
    let avatar = '';
    try { avatar = await sock.profilePictureUrl(jid, 'image'); } catch {}

    const dbStats = await getDBStats(instance);
    const status = getSyncStatus(instance);

    res.json({
        status: 'connected', 
        jid, 
        name: sock.authState.creds.me.name || instance,
        avatar, 
        contacts: dbStats.contacts,
        groups: dbStats.groups,
        messages: dbStats.messages,
        syncing: status.syncing,
        syncPhase: status.phase,
        syncCompleted: status.completed
    });
});

// ============================================
// ROTAS DE MENSAGENS
// ============================================

app.post('/v1/message/text', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance, number, text } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });

    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;
        const sent = await sock.sendMessage(jid, { text });
        await saveMessage(instance, jid, 'text', text);
        res.json({ status: 'success', key: sent.key });
    } catch(e) { 
        res.status(500).json({ error: e.message }); 
    }
});

app.post('/v1/message/buttons', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance, number, message, footer, buttons, title } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });

    const jid = number.includes('@') ? number : number + '@s.whatsapp.net';

    try {
        const result = await sendButtons(sock, jid, {
            title: title || '',
            text: message,
            footer: footer || 'NexusWA',
            buttons: buttons.map(b => ({ id: b.id, text: b.text }))
        });
        await saveMessage(instance, jid, 'buttons', JSON.stringify({ message, footer, title, buttons }));
        return res.json({ status: 'success', messageId: result?.key?.id || 'sent' });
    } catch (e) {
        return res.status(500).json({ error: e.message });
    }
});

app.post('/v1/message/list', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance, number, title, message, footer, buttonText, sections } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });

    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;
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
        await saveMessage(instance, jid, 'list', JSON.stringify({ title, message, footer, buttonText, sections }));
        res.json({ status: "success", messageId: result?.key?.id || 'sent' });
    } catch (e) {
        res.status(500).json({ error: e.message });
    }
});

app.post('/v1/message/url-button', authMiddleware, instanceAccessMiddleware, async (req, res) => {
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
        await saveMessage(instance, jid, 'url-button', JSON.stringify({ message, footer, title, buttonText, url }));
        res.json({ status: "success", messageId: result?.key?.id || 'sent' });
    } catch (e) {
        res.status(500).json({ error: e.message });
    }
});

app.post('/v1/message/copy-button', authMiddleware, instanceAccessMiddleware, async (req, res) => {
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
        await saveMessage(instance, jid, 'copy-button', JSON.stringify({ message, footer, title, buttonText, copyCode }));
        res.json({ status: "success", messageId: result?.key?.id || 'sent' });
    } catch (e) {
        res.status(500).json({ error: e.message });
    }
});

app.post('/v1/message/interactive', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance, number, interactive } = req.body;
    const sock = sessions.get(instance);
    if (!sock) return res.status(400).json({ error: 'Disconnected' });
    
    try {
        const jid = number.includes('@') ? number : `${number}@s.whatsapp.net`;
        
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
            await saveMessage(instance, jid, 'interactive', JSON.stringify(interactive));
            return res.json({ status: 'success', messageId: result?.key?.id || 'sent' });
        }
        
        const sent = await sock.sendMessage(jid, { 
            text: interactive.body?.text || interactive.text || 'Mensagem interativa'
        });
        res.json({ status: 'success', key: sent.key });
        
    } catch(e) { 
        res.status(500).json({ error: e.message }); 
    }
});

// ============================================
// ROTAS DE CONTATOS E GRUPOS
// ============================================

app.get('/v1/contacts/:instance', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    try {
        const contactsDB = await getContactsFromDB(instance);
        const groupsDB = await getGroupsFromDB(instance);
        
        const contacts = contactsDB.map(c => ({
            jid: c.jid,
            name: c.nome || c.jid.split('@')[0],
            is_group: false
        }));
        
        const groups = groupsDB.map(g => ({
            jid: g.jid,
            name: g.nome || g.jid.split('@')[0],
            is_group: true
        }));
        
        res.json([...contacts, ...groups]);
    } catch (e) {
        res.json([]);
    }
});

app.get('/v1/groups/:instance', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    try {
        const groupsDB = await getGroupsFromDB(instance);
        const result = groupsDB.map(g => ({
            jid: g.jid,
            name: g.nome || g.jid.split('@')[0],
            participants: 0,
            owner: '',
            created: g.criado_em
        }));
        res.json(result);
    } catch (e) {
        res.json([]);
    }
});

app.get('/v1/messages/:instance/:jid', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance, jid } = req.params;
    const storeData = getStore(instance);
    const msgs = Object.values(storeData.messages)
        .filter(m => m.key?.remoteJid === jid)
        .sort((a, b) => (a.messageTimestamp || 0) - (b.messageTimestamp || 0));
    res.json(msgs);
});

// ============================================
// ROTAS DE BANCO DE DADOS
// ============================================

app.get('/v1/db/contacts/:instance', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    const contacts = await getContactsFromDB(instance);
    res.json(contacts);
});

app.get('/v1/db/groups/:instance', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    const groups = await getGroupsFromDB(instance);
    res.json(groups);
});

app.get('/v1/db/messages/:instance/:jid', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance, jid } = req.params;
    const messages = await getMessagesFromDB(instance, jid);
    res.json(messages);
});

app.get('/v1/db/stats/:instance', authMiddleware, instanceAccessMiddleware, async (req, res) => {
    const { instance } = req.params;
    const stats = await getDBStats(instance);
    res.json(stats);
});

// ============================================
// ROTA DE INFO DO USUÁRIO ATUAL
// ============================================

app.get('/v1/me', authMiddleware, async (req, res) => {
    const apiKey = req.apiKeyData;
    res.json({
        id: apiKey.id,
        nome: apiKey.nome,
        tipo: apiKey.tipo,
        isSuperAdmin: req.isSuperAdmin,
        instanciasPermitidas: apiKey.instancias_permitidas,
        criadoEm: apiKey.criado_em,
        expiraEm: apiKey.expira_em,
        ultimoUso: apiKey.ultimo_uso
    });
});

// ============================================
// AUTO-RECOVERY E INICIALIZAÇÃO
// ============================================

async function recoverSessions() {
    const instances = await getAllInstances();
    
    for (const inst of instances) {
        const authPath = `auth_info/${inst.nome}`;
        if (fs.existsSync(authPath)) {
            console.log(`Recuperando sessão: ${inst.nome}`);
            startSession(inst.nome);
        }
    }
}

async function init() {
    if (!fs.existsSync('auth_info')) fs.mkdirSync('auth_info');
    
    // Garante que existe um Super Admin
    await ensureSuperAdminExists();
    
    // Recupera sessões
    await recoverSessions();
    
    app.listen(PORT, () => console.log(`🚀 Nexus Baileys API rodando na porta ${PORT}`));
}

init();