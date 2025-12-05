// buttons.js — suporte REAL para botões usando Native Flow (funciona no app oficial WhatsApp)
// Este arquivo é mantido para compatibilidade, mas as funções principais estão no index.js

const {
    generateWAMessageFromContent,
    proto
} = require('@whiskeysockets/baileys');

/**
 * Envia botões usando Native Flow Message (funciona no WhatsApp oficial)
 * @param {object} sock - Socket do Baileys
 * @param {string} jid - JID do destinatário
 * @param {object} options - Opções { text, footer, buttons, title }
 */
async function sendButtons(sock, jid, options) {
    const { text, footer, buttons, title } = options;

    // Converte botões para formato nativeFlowMessage
    const nativeButtons = buttons.map(btn => ({
        name: "quick_reply",
        buttonParamsJson: JSON.stringify({
            display_text: btn.text || btn.displayText || btn.title,
            id: btn.id || btn.buttonId || `btn_${Date.now()}`
        })
    }));

    const msg = generateWAMessageFromContent(jid, {
        viewOnceMessage: {
            message: {
                messageContextInfo: {
                    deviceListMetadata: {},
                    deviceListMetadataVersion: 2
                },
                interactiveMessage: proto.Message.InteractiveMessage.create({
                    body: proto.Message.InteractiveMessage.Body.create({
                        text: text || "Selecione uma opção"
                    }),
                    footer: proto.Message.InteractiveMessage.Footer.create({
                        text: footer || "NexusWA"
                    }),
                    header: proto.Message.InteractiveMessage.Header.create({
                        title: title || "",
                        subtitle: "",
                        hasMediaAttachment: false
                    }),
                    nativeFlowMessage: proto.Message.InteractiveMessage.NativeFlowMessage.create({
                        buttons: nativeButtons
                    })
                })
            }
        }
    }, {});

    return await sock.relayMessage(jid, msg.message, { messageId: msg.key.id });
}

/**
 * Envia lista de seleção usando Native Flow Message
 */
async function sendList(sock, jid, options) {
    const { text, footer, title, buttonText, sections } = options;

    const listButton = {
        name: "single_select",
        buttonParamsJson: JSON.stringify({
            title: buttonText || "Selecionar",
            sections: sections.map(section => ({
                title: section.title || "Opções",
                highlight_label: section.highlight || "",
                rows: (section.rows || []).map(row => ({
                    header: row.header || "",
                    title: row.title || row.text,
                    description: row.description || "",
                    id: row.id || row.rowId || `row_${Date.now()}`
                }))
            }))
        })
    };

    const msg = generateWAMessageFromContent(jid, {
        viewOnceMessage: {
            message: {
                messageContextInfo: {
                    deviceListMetadata: {},
                    deviceListMetadataVersion: 2
                },
                interactiveMessage: proto.Message.InteractiveMessage.create({
                    body: proto.Message.InteractiveMessage.Body.create({
                        text: text || "Selecione uma opção"
                    }),
                    footer: proto.Message.InteractiveMessage.Footer.create({
                        text: footer || "NexusWA"
                    }),
                    header: proto.Message.InteractiveMessage.Header.create({
                        title: title || "",
                        subtitle: "",
                        hasMediaAttachment: false
                    }),
                    nativeFlowMessage: proto.Message.InteractiveMessage.NativeFlowMessage.create({
                        buttons: [listButton]
                    })
                })
            }
        }
    }, {});

    return await sock.relayMessage(jid, msg.message, { messageId: msg.key.id });
}

/**
 * Envia botão com URL (Call to Action)
 */
async function sendUrlButton(sock, jid, options) {
    const { text, footer, title, buttonText, url } = options;

    const urlButton = {
        name: "cta_url",
        buttonParamsJson: JSON.stringify({
            display_text: buttonText || "Acessar",
            url: url,
            merchant_url: url
        })
    };

    const msg = generateWAMessageFromContent(jid, {
        viewOnceMessage: {
            message: {
                messageContextInfo: {
                    deviceListMetadata: {},
                    deviceListMetadataVersion: 2
                },
                interactiveMessage: proto.Message.InteractiveMessage.create({
                    body: proto.Message.InteractiveMessage.Body.create({
                        text: text || ""
                    }),
                    footer: proto.Message.InteractiveMessage.Footer.create({
                        text: footer || "NexusWA"
                    }),
                    header: proto.Message.InteractiveMessage.Header.create({
                        title: title || "",
                        subtitle: "",
                        hasMediaAttachment: false
                    }),
                    nativeFlowMessage: proto.Message.InteractiveMessage.NativeFlowMessage.create({
                        buttons: [urlButton]
                    })
                })
            }
        }
    }, {});

    return await sock.relayMessage(jid, msg.message, { messageId: msg.key.id });
}

/**
 * Envia botão de copiar código
 */
async function sendCopyButton(sock, jid, options) {
    const { text, footer, title, buttonText, copyCode } = options;

    const copyButton = {
        name: "cta_copy",
        buttonParamsJson: JSON.stringify({
            display_text: buttonText || "Copiar",
            copy_code: copyCode
        })
    };

    const msg = generateWAMessageFromContent(jid, {
        viewOnceMessage: {
            message: {
                messageContextInfo: {
                    deviceListMetadata: {},
                    deviceListMetadataVersion: 2
                },
                interactiveMessage: proto.Message.InteractiveMessage.create({
                    body: proto.Message.InteractiveMessage.Body.create({
                        text: text || ""
                    }),
                    footer: proto.Message.InteractiveMessage.Footer.create({
                        text: footer || "NexusWA"
                    }),
                    header: proto.Message.InteractiveMessage.Header.create({
                        title: title || "",
                        subtitle: "",
                        hasMediaAttachment: false
                    }),
                    nativeFlowMessage: proto.Message.InteractiveMessage.NativeFlowMessage.create({
                        buttons: [copyButton]
                    })
                })
            }
        }
    }, {});

    return await sock.relayMessage(jid, msg.message, { messageId: msg.key.id });
}

module.exports = { 
    sendButtons, 
    sendList, 
    sendUrlButton, 
    sendCopyButton 
};