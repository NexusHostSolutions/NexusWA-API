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
