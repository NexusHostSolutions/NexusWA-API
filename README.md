<p align="center"> <img src="https://via.placeholder.com/900x200/020617/0ea5e9?text=NexusWA+API" style="border-radius:10px" /> </p> <h1 align="center">⚡ NexusWA-API</h1> <p align="center">API WhatsApp multi-instância para integrações, automação, atendimentos e sistemas de comunicação.</p> <p align="center"> <img src="https://img.shields.io/badge/Status-Ativo-brightgreen?style=for-the-badge"> <img src="https://img.shields.io/badge/Framework-Node.js-black?style=for-the-badge&logo=node.js"> <img src="https://img.shields.io/badge/Backend-Go-blue?style=for-the-badge&logo=go"> <img src="https://img.shields.io/badge/API-REST%20JSON-orange?style=for-the-badge"> </p>
📌 Sobre o Projeto

O NexusWA-API é uma API de comunicação automatizada para WhatsApp com:

✔ Gerenciamento multi-instância
✔ Envio de mensagens programáticas
✔ Sessões persistentes com reconexão automática
✔ Consulta de contatos, grupos e mensagens recentes
✔ Integração com painéis, bots, CRMs e automações

Ideal para empresas, provedores de automação, suporte, SAC 24/7 e integrações avançadas.

📦 Instalação
1. Clonar o projeto
git clone https://github.com/NexusHostSolutions/NexusWA-API.git
cd NexusWA-API

2. Configurar dependências Node
cd nex-buttons
npm install

3. Backend Go (opcional)
go mod tidy

▶ Executar o servidor
node nex-buttons/index.js


Servidor iniciado em:

http://localhost:3001

🔐 Sessões WhatsApp
📍 Criar sessão (QR ou Pareamento)
curl -X POST http://localhost:3001/session/start \
-H "Content-Type: application/json" \
-d '{"instance":"minhaSessao"}'

Pareamento com número
curl -X POST http://localhost:3001/session/pair-code \
-H "Content-Type: application/json" \
-d '{"instance":"minhaSessao","phoneNumber":"559999999999"}'

💬 Enviar mensagem
curl -X POST http://localhost:3001/v1/message/text \
-H "Content-Type: application/json" \
-d '{
  "instance":"minhaSessao",
  "number":"559999999999",
  "text":"Olá! 😊"
}'

📇 Contatos & Grupos
Listar contatos
curl http://localhost:3001/v1/contacts/minhaSessao

Listar grupos
curl http://localhost:3001/v1/groups/minhaSessao

📁 Estrutura
📂 NexusWA-API
 ├─ 📂 nex-buttons        → Núcleo responsável pelas sessões
 ├─ 📂 internal           → Backend Go complementar
 ├─ 📂 auth_info          → Tokens da sessão (não público)
 ├─ README.md
 └─ .gitignore

🔥 Capturas do Projeto

Você poderá colocar imagens reais aqui futuramente

Tela	Preview
QR Code de conexão	

Lista de Contatos	

Instâncias conectadas	
🔥 Roadmap
Feature	Status
Webhook mensagens recebidas	🚧 Em desenvolvimento
Banco de dados para contatos	🔜
Envio de mídia	🔜
API Token Security	🔜
Painel administrativo moderno	🔥 Previsto
Deploy com Docker	🔥 Previsto
📜 Licença

Uso autorizado apenas pelo proprietário/revenda.
Distribuição comercial externa requer permissão.

👨‍💻 Desenvolvido por

NexusHost Solutions
Soluções profissionais em automação & integração para WhatsApp.

📩 suporte@nexushostsolutions.com.br

🌐 https://nexushostsolutions.com.br