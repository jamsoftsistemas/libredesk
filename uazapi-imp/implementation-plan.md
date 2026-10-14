# Plano: canal UAZAPI (WhatsApp não oficial) no Libredesk

## Context
O libredesk só tem o canal `whatsapp` (Meta Cloud API). Queremos um novo canal `uazapi` que use o gateway UAZAPI (`uazapi-openapi-spec.yaml`, v2.4.4): conexão por QR code, envio/recebimento de texto e mídia, status de entrega/leitura. Reaproveitamos a arquitetura de inbox plugável, `streamqueue`, `keyedLock` e as APIs de contato/conversa; o client, o parser de webhook e a autenticação são novos (UAZAPI não tem HMAC, nem templates, nem janela de 24h).

Escopo MVP: texto, mídia (imagem/vídeo/áudio/documento), reply/quote, status delivered/read, conexão via QR, desconexão. Fora do escopo: grupos, newsletters, botões/menus/carrossel, pagamentos, templates, CSAT por template.

Decisões: identidade de contato separada (`uazapi` no enum `channels`); base URL configurável (cloud `free`/`api` ou self-hosted); webhook autenticado por segredo no path/query (a spec não oferece assinatura).

## Fase 1 — Banco e constantes
- `schema.sql:3`: adicionar `'uazapi'` ao enum `channels`.
- Nova migration `internal/migrations/vX.Y.Z.go` (modelo: `v2.9.0.go`, `ALTER TYPE channels ADD VALUE IF NOT EXISTS 'uazapi'`, statement isolado) e registrar em `cmd/upgrade.go`.
- `internal/inbox/inbox.go:28-32` e `inbox/models/models.go:149`: constante `ChannelUazapi`.

## Fase 2 — Client e canal (`internal/uazapi/` + `internal/inbox/channel/uazapi/`)
- `internal/uazapi/client.go`: HTTP client (headers `token` / `admintoken`, timeout, tratamento de 429 com `Retry-After`, erro 463 `REACHOUT_TIMELOCK` como erro tipado). Métodos: `CreateInstance`, `Connect`, `Status`, `Disconnect`, `SetWebhook`, `SendText`, `SendMedia`, `DownloadMedia`, `MarkRead`.
- `internal/uazapi/webhook.go`: structs e parser para `messages`, `messages_update` (estados Delivered/Read/Played, só avançar status), `connection` (Disconnected/TemporaryBan).
- `internal/inbox/channel/uazapi/uazapi.go`: espelha `channel/whatsapp/whatsapp.go` — `Config{BaseURL, InstanceToken, AdminToken, WebhookSecret, ...}`, `Send` (texto → `/send/text`, anexo → `/send/media` com tipo `image|video|document|audio|ptt`, `replyid` para quote, `track_source=libredesk` + `track_id`), persistência do `messageid` via `SourceIDUpdater`, retry só em 429/5xx, `Receive` no-op.

## Fase 3 — Registro, segredos e validação (backend `cmd/` + `internal/inbox`)
- `cmd/init.go` (~820-835): caso `uazapi` em `makeInboxInitializer`; `reloadInbox`/`startInboxes`.
- `inbox.go` `encryptInboxConfig`/`decryptInboxConfig` (~639/712), `models.go` `ClearPasswords` (:149), merge de segredos no update (`MergeWhatsAppSecrets` :462 → generalizar ou duplicar): campos `instance_token`, `admin_token`, `webhook_secret`.
- `cmd/inboxes.go`: `validateInbox` (:450), `trimInboxFields` (:~765), hook pós-save que registra o webhook na UAZAPI (`POST /webhook`, events `messages`, `messages_update`, `connection`; `excludeMessages: [wasSentByApi, isGroupYes]` — validar com tráfego real que status de mensagens enviadas ainda chega) e hook de delete.
- Endpoints novos autenticados (perm de admin de inbox): `POST /api/v1/inboxes/{id}/uazapi/connect`, `GET .../uazapi/status` (retorna status + qrcode/paircode), `POST .../uazapi/disconnect`. Registrar em `cmd/handlers.go`.

## Fase 4 — Webhook e ingestão
- `cmd/handlers.go` (~401-413): `POST /webhooks/uazapi/{inbox_id}/{secret}` sem `auth()`, com rate limit; comparar segredo em tempo constante.
- `cmd/uazapi_webhook.go` + `cmd/uazapi_ingester.go`: handler enfileira o body e responde 200 rápido; ingester sobre `internal/streamqueue` (stream `libredesk:uazapi:inbound`, `keyedLock` por remetente) — modelo `cmd/whatsapp_ingester.go`. `ensureUazapiIngester` análogo a `ensureWhatsAppIngester` (`cmd/main.go:145` guarda o estado).
- Ingestão (modelo `ingestWhatsAppMessage`, `cmd/whatsapp_webhook.go:179`): ignorar `fromMe`/grupos; dedupe por `messageid`; `UpsertContactByChannelIdentity("uazapi", número)`; `SetContactPhoneIfMissing`; reabrir/criar conversa (`GetLatestOpenConversationForContact`, `GetReopenableConversationForContact`, `CreateConversation`); baixar mídia via `/message/download` (fileURL vale 2 dias, baixar na hora, respeitar `MaxFileUploadSizeMB`, placeholder em falha permanente); `ProcessIncomingMessage`.
- Status: `messages_update` → atualizar `provider_status` (reusar `ApplyWhatsAppStatus`/query `apply-whatsapp-message-status`, `queries.sql:1017`, generalizando o filtro de canal).
- Evento `connection`: logar e expor no status da inbox (alerta em `TemporaryBan`).

## Fase 5 — Camada de conversa (`internal/conversation`)
Generalizar checagens hard-coded de `whatsapp`:
- `message.go:186` (falha de envio), `:318` (**obrigatório**: canal desconhecido é rejeitado), `:611/:645` (switch de outbound) → `prepareUazapiOutbound`: resolve telefone do contato (identidade `uazapi`, fallback telefone + `DialCodeForISO`), grava `meta["uazapi"]`; sem validação de template nem janela de 24h.
- `queries.sql:992,1048` e `cmd/conversation.go:459,880,981,1065,1148` (read receipt, criação de conversa por agente sem template). `markread` best-effort via `/message/markread`.
- `models/models.go:456`: ler `meta["uazapi"]`.
- Sem CSAT por template (`conversation.go:1894` ignora uazapi; CSAT por link web continua).

## Fase 6 — Frontend (`frontend/apps/main/src`)
- `views/admin/inbox/NewInbox.vue` (card + submit `channel:'uazapi'`), `EditInbox.vue`, `InboxList.vue:103` (mapa de i18n), `constants/channelIcons.js`.
- `features/admin/inbox/UazapiInboxForm.vue` + `uazapiFormSchema.js` (zod): name, enabled, base_url, instance_token/admin_token, reopen_window_hours, csat_enabled. Segredos mascarados como no form WhatsApp.
- Tela/diálogo de conexão: QR + paircode com polling a cada 1–2 s em `/uazapi/status` (padrão de `components/importer/Importer.vue:203-216`), estados connecting/connected/disconnected, botão desconectar.
- `api/index.js` (~409-441): `connectUazapi`, `getUazapiStatus`, `disconnectUazapi`.
- Helper de capacidades por canal (hoje `WHATSAPP_CHANNEL` fixo em `whatsappTemplate.js`): uazapi usa o composer normal **sem** banner de 24h e **sem** botão de template; anexos aceitam mídia ampla e um por mensagem com legenda. Ajustar `stores/inbox.js:19`, `CreateConversation.vue`, `messageDeliveryStatus.js:4`, `ReplyBoxMenuBar.vue`, `ReplyBox.vue`.
- i18n: `i18n/*.json` (chaves planas e ordenadas; `pt-BR` e `en-US` completos, demais com fallback em inglês): `admin.inbox.uazapi.*`, `admin.inbox.help.uazapi`, `conversation.uazapi.*`, `globals.terms.uazapi`.

## Fase 7 — Testes
- Go: client (httptest), parser de webhook com payloads reais, ingestão (dedupe, mídia, fromMe/grupo), status só avança, envio texto/mídia/reply, 429/463 — modelos: `whatsapp_test.go`, `whatsapp_ingester_test.go`, `whatsapp_webhook_test.go`.
- Cypress: `frontend/cypress/e2e/ui/uazapiInboxForm.cy.js` (modelo `whatsappInboxForm.cy.js`).

## Verificação
1. `go build ./... && go test ./...`; no frontend `pnpm lint` e os testes unitários/Cypress.
2. Subir o stack (docker compose), rodar a migration; criar inbox uazapi com instância de teste (servidor `free`) e escanear o QR; confirmar status `connected`.
3. Enviar mensagem de um celular → conversa criada com contato `uazapi`; responder texto, imagem, documento e áudio (ptt) pelo painel; conferir ticks delivered/read.
4. Reenviar o mesmo webhook (dedupe), derrubar o Redis/ingester (retry), desconectar instância (evento `connection`), e forçar 429.
5. Confirmar que inboxes `whatsapp` (Meta) continuam funcionando (regressão das queries generalizadas).

## Riscos
- Conta fora do WhatsApp Business gera desconexões; erro 463 (timelock de novos chats) precisa aparecer legível ao agente.
- Formato real de `messageType`/`content` de mídia não está na spec — capturar payloads reais antes de fechar o parser.
- `excludeMessages: wasSentByApi` pode filtrar status das mensagens enviadas pela API; validar e, se preciso, remover o filtro e ignorar eco no ingester.
- Sem assinatura de webhook: segredo no path + rate limit; não logar a URL completa.

## Estimativa
~3–3,5 semanas (1 dev): fases 1–3 ≈ 1 sem; 4–5 ≈ 1 sem; 6 ≈ 0,5–1 sem; 7 + ajustes ≈ 0,5 sem.
