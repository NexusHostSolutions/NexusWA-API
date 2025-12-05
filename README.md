# ⚡ NexusWA-API
API WhatsApp multi-instância para automação, integrações e atendimento inteligente.

🚀 Ideal para bots, CRMs, suporte, provedores e automações em massa.

---

## 🔰 Instalação

```bash
git clone https://github.com/NexusHostSolutions/NexusWA-API.git
cd NexusWA-API
```

### Node (API principal)
```bash
cd nex-buttons
npm install
node index.js
```

### Go Backend (opcional)
```bash
go mod tidy
go run cmd/server/main.go
```

🌐 Servidor padrão: http://localhost:3001

---

## 🔐 Gerenciar Sessão

Criar sessão (QR Code / Pair)

```bash
curl -X POST http://localhost:3001/session/start \
-H "Content-Type: application/json" \
-d '{"instance":"minhaSessao"}'
```

Pareamento com número:

```bash
curl -X POST http://localhost:3001/session/pair-code \
-H "Content-Type: application/json" \
-d '{"instance":"minhaSessao","phoneNumber":"559999999999"}'
```

---

## 💬 Enviar mensagem

```bash
curl -X POST http://localhost:3001/v1/message/text \
-H "Content-Type: application/json" \
-d '{ "instance":"minhaSessao", "number":"559999999999", "text":"Olá! 👋" }'
```

---

## 📇 Contatos & Grupos

```bash
curl http://localhost:3001/v1/contacts/minhaSessao
curl http://localhost:3001/v1/groups/minhaSessao
```

---

## 📁 Estrutura do Projeto

```
📂 NexusWA-API
├─ 📂 nex-buttons   → Core API WhatsApp
├─ 📂 internal      → Go backend extra
├─ 📂 docs          → Interface Documentação
├─ 📂 auth_info     → Sessões (Não versionar)
├─ README.md
└─ .gitignore
```

---

## 🔥 Roadmap

| Feature | Status |
|---|---|
| Multi-instância | ✔ |
| Botões Interativos | ✔ |
| Lista interativa | ✔ |
| Contatos & grupos API | ✔ |
| Webhooks | 🚧 |
| Banco de contatos | 🔜 |
| Envio de mídia | 🔜 |
| Painel admin completo | 🔥 Futuro update |

---

## 👨‍💻 Desenvolvido por
**NexusHost Solutions**  
🌐 https://nexushostsolutions.com.br  
📩 suporte@nexushostsolutions.com.br

---

---

# 📄 Interface de Documentação  
Crie o arquivo:

📁 `docs/index.html`

Cole dentro exatamente o código abaixo:

```html
<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<title>NexusWA API Docs</title>
<style>
body{
    background:#0d0d0d;
    font-family:Arial, sans-serif;
    color:#fff;
    margin:0;
}
.container{
    max-width:850px;
    margin:auto;
    padding:40px;
    background:rgba(255,255,255,0.05);
    backdrop-filter:blur(10px);
    border-radius:16px;
    margin-top:40px;
    border:1px solid rgba(255,255,255,0.1);
}
h1{font-size:32px; color:#00ff88; text-align:center;}
code,pre{background:#111;padding:12px;border-radius:8px;color:#00ff88;display:block;}
.btn{
    background:#00ff88;padding:12px 22px;color:#000;
    border-radius:6px;text-decoration:none;font-weight:bold;
}
.section{margin-top:30px;}
.footer{text-align:center;margin-top:50px;color:#888;}
hr{border-color:#222;}
.center{text-align:center;}
.logo{
    width:100px;display:block;margin:auto;margin-bottom:20px;
    filter:drop-shadow(0px 0px 6px #00ff88);
}
</style>
</head>

<body>
<div class="container">

<img src="https://upload.wikimedia.org/wikipedia/commons/6/6b/WhatsApp.svg" class="logo">

<h1>📘 NexusWA - API Documentation</h1>

<p>API WhatsApp multi-instância para automação, bots e integrações profissionais.</p>

<hr>

<div class="section">
<h2>🚀 Iniciar Sessão</h2>
<pre>POST /session/start {"instance":"Nexus01"}</pre>
<pre>POST /session/pair-code {"instance":"Nexus01","phoneNumber":"559999999999"}</pre>
</div>

<div class="section">
<h2>💬 Enviar mensagem</h2>
<pre>POST /v1/message/text {"instance":"Nexus01","number":"559999999999","text":"Olá!"}</pre>
</div>

<div class="section">
<h2>📇 Contatos / Grupos</h2>
<pre>GET /v1/contacts/Nexus01</pre>
<pre>GET /v1/groups/Nexus01</pre>
</div>

<div class="section center">
<a href="https://nexushostsolutions.com.br" class="btn">Site Oficial</a>
</div>

<div class="footer">
<hr>
© 2025 NexusHost Solutions - Todos os direitos reservados.
</div>
</div>
</body>
</html>
