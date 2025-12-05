<!DOCTYPE html>
<html lang="pt-br">
<head>
<meta charset="UTF-8">
<title>NexusWA-API Documentation</title>
<meta name="viewport" content="width=device-width, initial-scale=1">

<style>
    body{
        font-family: Arial, sans-serif;
        background:#0d1117;
        color:#e6edf3;
        margin:0;
        padding:0;
        line-height:1.6;
    }
    header{
        background:#00a884;
        color:#fff;
        padding:30px;
        text-align:center;
        font-size:32px;
        font-weight:bold;
        letter-spacing:2px;
        display:flex;
        justify-content:center;
        align-items:center;
        gap:15px;
    }
    header img{
        width:60px;
        filter:drop-shadow(0px 0px 6px #00000060);
    }
    .container{
        max-width:900px;
        margin:auto;
        padding:25px;
    }
    h2{
        color:#00a884;
        border-left:5px solid #00a884;
        padding-left:10px;
        margin-top:35px;
    }
    code, pre{
        background:#161b22;
        color:#00ff9d;
        padding:10px;
        display:block;
        border-radius:6px;
        overflow-x:auto;
    }
    .box{
        background:#11161d;
        padding:18px;
        border-radius:8px;
        margin-top:15px;
        border:1px solid #1f2937;
    }
    .list-check span{display:block;margin-bottom:6px;}
    .list-check span::before{
        content:"✔ ";
        color:#00ff9d;
    }
    footer{
        margin-top:50px;
        text-align:center;
        padding:20px;
        background:#00a884;
        color:#fff;
        font-weight:bold;
    }
    table{
        width:100%;
        margin-top:15px;
        border-collapse:collapse;
    }
    table td, table th{
        border:1px solid #333;
        padding:10px;
        text-align:center;
    }
    th{
        background:#00a884;
        color:#000;
    }
</style>
</head>

<body>

<header>
    <img src="https://upload.wikimedia.org/wikipedia/commons/6/6b/WhatsApp.svg"
         alt="WhatsApp Logo">
    ⚡ NexusWA-API
</header>

<div class="container">

<p>API WhatsApp multi-instância para integrações, automação, atendimentos e sistemas de comunicação.</p>

<h2>📌 Sobre o Projeto</h2>

<p>O <b>NexusWA-API</b> é uma API de comunicação automatizada para WhatsApp com:</p>

<div class="box list-check">
<span>Gerenciamento multi-instância</span>
<span>Envio de mensagens programáticas</span>
<span>Sessões persistentes com reconexão automática</span>
<span>Consulta de contatos, grupos e mensagens recentes</span>
<span>Integração com automações, CRMs e sistemas externos</span>
</div>

<p><i>Ideal para empresas, provedores de automação, suporte, SAC 24/7 e integrações avançadas.</i></p>

<h2>📦 Instalação</h2>

<pre><code>git clone https://github.com/NexusHostSolutions/NexusWA-API.git
cd NexusWA-API
</code></pre>

<b>Instalar dependências Node</b>
<pre><code>cd nex-buttons
npm install
</code></pre>

<b>Backend Go (opcional)</b>
<pre><code>go mod tidy
</code></pre>

<h2>▶ Executar o servidor</h2>

<pre><code>node nex-buttons/index.js
</code></pre>

Servidor iniciado em:

<pre><code>http://localhost:3001
</code></pre>

<h2>🔐 Sessões WhatsApp</h2>

<pre><code>curl -X POST http://localhost:3001/session/start \
-H "Content-Type: application/json" \
-d '{"instance":"minhaSessao"}'
</code></pre>

<pre><code>curl -X POST http://localhost:3001/session/pair-code \
-H "Content-Type: application/json" \
-d '{"instance":"minhaSessao","phoneNumber":"559999999999"}'
</code></pre>

<h2>💬 Enviar mensagem</h2>
<pre><code>curl -X POST http://localhost:3001/v1/message/text \
-H "Content-Type: application/json" \
-d '{ "instance":"minhaSessao", "number":"559999999999", "text":"Olá! 😊" }'
</code></pre>

<h2>📇 Contatos & Grupos</h2>

<pre><code>curl http://localhost:3001/v1/contacts/minhaSessao
curl http://localhost:3001/v1/groups/minhaSessao
</code></pre>


<h2>📁 Estrutura</h2>

<pre><code>📂 NexusWA-API
 ├─ 📂 nex-buttons
 ├─ 📂 internal
 ├─ 📂 auth_info
 └─ README.md
</code></pre>


<h2>🔥 Capturas (em breve)</h2>

<table>
<tr><th>Tela</th><th>Preview</th></tr>
<tr><td>QR Code</td><td>📷</td></tr>
<tr><td>Contatos</td><td>📄</td></tr>
<tr><td>Instâncias</td><td>⚙</td></tr>
</table>

<h2>📜 Licença</h2>
<p>Uso autorizado ao proprietário. Distribuição comercial requer permissão.</p>

<h2>👨‍💻 Desenvolvido por</h2>
<p><b>NexusHost Solutions</b><br>Automação & infraestrutura WhatsApp.<br><br>
📩 suporte@nexushostsolutions.com.br<br>
🌐 https://nexushostsolutions.com.br</p>

</div>

<footer>NexusWA-API — Todos os direitos reservados</footer>
</body>
</html>
