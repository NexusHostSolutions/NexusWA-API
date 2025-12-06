-- ============================================
-- NexusWA-API - Schema do Banco de Dados
-- Execute este script no PostgreSQL
-- ============================================

-- Tabela de Contatos
CREATE TABLE IF NOT EXISTS contatos (
    id SERIAL PRIMARY KEY,
    instance VARCHAR(100) NOT NULL,
    jid VARCHAR(100) NOT NULL,
    nome VARCHAR(255),
    criado_em TIMESTAMP DEFAULT NOW(),
    UNIQUE(instance, jid)
);

-- Tabela de Grupos
CREATE TABLE IF NOT EXISTS grupos (
    id SERIAL PRIMARY KEY,
    instance VARCHAR(100) NOT NULL,
    jid VARCHAR(100) NOT NULL,
    nome VARCHAR(255),
    criado_em TIMESTAMP DEFAULT NOW(),
    UNIQUE(instance, jid)
);

-- Tabela de Mensagens
CREATE TABLE IF NOT EXISTS mensagens (
    id SERIAL PRIMARY KEY,
    instance VARCHAR(100) NOT NULL,
    jid VARCHAR(100) NOT NULL,
    tipo VARCHAR(50) NOT NULL,
    conteudo TEXT,
    criado_em TIMESTAMP DEFAULT NOW()
);

-- Índices para melhor performance
CREATE INDEX IF NOT EXISTS idx_contatos_instance ON contatos(instance);
CREATE INDEX IF NOT EXISTS idx_grupos_instance ON grupos(instance);
CREATE INDEX IF NOT EXISTS idx_mensagens_instance_jid ON mensagens(instance, jid);
CREATE INDEX IF NOT EXISTS idx_mensagens_criado_em ON mensagens(criado_em DESC);


-- ============================================
-- SISTEMA DE API KEYS COM PERMISSÕES
-- ============================================

-- Tipos de usuário: 'super_admin', 'user'
-- Validades: 30, 90, 180, 365, NULL (nunca expira)

CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    key_hash VARCHAR(64) UNIQUE NOT NULL,      -- Hash SHA256 da key
    key_prefix VARCHAR(10) NOT NULL,            -- Primeiros 8 chars para identificação
    nome VARCHAR(100) NOT NULL,                 -- Nome descritivo
    tipo VARCHAR(20) DEFAULT 'user',            -- 'super_admin' ou 'user'
    ativa BOOLEAN DEFAULT TRUE,
    instancias_permitidas TEXT[],               -- Array de instâncias que pode acessar (NULL = todas para admin)
    criado_em TIMESTAMP DEFAULT NOW(),
    expira_em TIMESTAMP,                        -- NULL = nunca expira
    ultimo_uso TIMESTAMP,
    total_requisicoes INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS instancias (
    id SERIAL PRIMARY KEY,
    nome VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(50) DEFAULT 'disconnected',
    push_name VARCHAR(255),
    jid VARCHAR(100),
    avatar TEXT,
    webhook_url TEXT,
    webhook_enabled BOOLEAN DEFAULT FALSE,
    criado_por INTEGER REFERENCES api_keys(id), -- Quem criou
    criado_em TIMESTAMP DEFAULT NOW(),
    atualizado_em TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contatos (
    id SERIAL PRIMARY KEY,
    instance VARCHAR(100),
    jid VARCHAR(100),
    nome VARCHAR(255),
    criado_em TIMESTAMP DEFAULT NOW(),
    UNIQUE(instance, jid)
);

CREATE TABLE IF NOT EXISTS grupos (
    id SERIAL PRIMARY KEY,
    instance VARCHAR(100),
    jid VARCHAR(100),
    nome VARCHAR(255),
    criado_em TIMESTAMP DEFAULT NOW(),
    UNIQUE(instance, jid)
);

CREATE TABLE IF NOT EXISTS mensagens (
    id SERIAL PRIMARY KEY,
    instance VARCHAR(100),
    jid VARCHAR(100),
    tipo VARCHAR(50),
    conteudo TEXT,
    criado_em TIMESTAMP DEFAULT NOW()
);

-- Logs de acesso
CREATE TABLE IF NOT EXISTS logs_acesso (
    id SERIAL PRIMARY KEY,
    api_key_id INTEGER REFERENCES api_keys(id),
    endpoint VARCHAR(255),
    metodo VARCHAR(10),
    ip VARCHAR(50),
    status_code INTEGER,
    criado_em TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_tipo ON api_keys(tipo);
CREATE INDEX IF NOT EXISTS idx_instancias_nome ON instancias(nome);
CREATE INDEX IF NOT EXISTS idx_contatos_instance ON contatos(instance);
CREATE INDEX IF NOT EXISTS idx_grupos_instance ON grupos(instance);
CREATE INDEX IF NOT EXISTS idx_mensagens_instance_jid ON mensagens(instance, jid);
CREATE INDEX IF NOT EXISTS idx_logs_acesso_key ON logs_acesso(api_key_id);

-- ============================================
-- CRIAR SUPER ADMIN PADRÃO (primeira execução)
-- ============================================
-- A key será: nexus_superadmin_XXXXXXXX (gerada no Node.js)