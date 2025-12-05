#!/bin/bash

# =============================
# 🔐 CONFIGURAÇÃO DE SEGURANÇA
# =============================
SENHA_CORRETA="NEXUS-KEY-2025"   # <<< ALTERE A SENHA AQUI
PRODUTO="NexusWA-API"
EMPRESA="NexusHost Solutions"

clear
echo "=============================================================="
echo " 🚀 Instalador Oficial - $PRODUTO"
echo " 📌 Desenvolvido por: $EMPRESA"
echo "=============================================================="
echo ""
echo "🔒 Este instalador requer uma chave de ativação."
echo -n "Digite sua chave de instalação: "
read SENHA_DIGITADA

if [ "$SENHA_DIGITADA" != "$SENHA_CORRETA" ]; then
    echo ""
    echo "❌ Chave incorreta! A instalação foi bloqueada."
    echo "Entre em contato para adquirir acesso:"
    echo "📩 suporte@nexushostsolutions.com.br"
    echo "🌐 https://nexushostsolutions.com.br"
    exit 1
fi

echo ""
echo "✔ Chave válida! Continuando com a instalação..."
sleep 1

# =============================
# 1. Atualização de pacotes
# =============================
echo ""
echo "📦 Atualizando sistema..."
sudo apt update -y && sudo apt upgrade -y

# =============================
# 2. Dependências essenciais
# =============================
echo ""
echo "⚙ Instalando dependências..."
sudo apt install -y curl git build-essential

# =============================
# 3. Instalando NodeJS & npm
# =============================
echo ""
echo "🟢 Instalando NodeJS..."
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt install -y nodejs npm

# =============================
# 4. Instalando PM2
# =============================
echo ""
echo "🟢 Instalando PM2 (Para rodar 24/7)..."
sudo npm install -g pm2

# =============================
# 5. Instalando Go (opcional)
# =============================
echo ""
echo "🟢 Instalando Go..."
sudo apt install -y golang

# =============================
# 6. Clonando repositório
# =============================
echo ""
echo "📥 Baixando projeto oficial do GitHub..."
git clone https://github.com/NexusHostSolutions/NexusWA-API.git
cd NexusWA-API/nex-buttons

echo ""
echo "📦 Instalando pacotes..."
npm install

# =============================
# 7. Preparando estrutura
# =============================
mkdir -p auth_info
mkdir -p logs

# =============================
# 8. Iniciando o serviço
# =============================
echo ""
echo "🚀 Iniciando API com PM2..."
pm2 start index.js --name nexuswa-api
pm2 save
pm2 startup systemd -u $USER --hp $HOME > /dev/null

sleep 1
clear

# =============================
# 9. Finalização
# =============================
echo "=============================================================="
echo "       ✅ INSTALAÇÃO CONCLUÍDA COM SUCESSO!"
echo "=============================================================="
echo ""
echo "📌 Produto: $PRODUTO"
echo "🏷 Empresa: $EMPRESA"
echo "🌍 Acesse sua API: http://SEU_IP:3001"
echo ""
echo "📄 Comandos úteis:"
echo "   🔹 pm2 logs nexuswa-api"
echo "   🔹 pm2 restart nexuswa-api"
echo "   🔹 pm2 stop nexuswa-api"
echo ""
echo "📩 Suporte técnico: suporte@nexushostsolutions.com.br"
echo "🌐 Website: https://nexushostsolutions.com.br"
echo ""
echo "=============================================================="
echo " Obrigado por utilizar soluções oficiais da $EMPRESA!"
echo "=============================================================="
