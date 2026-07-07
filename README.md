# Hysteria 2 Protocol-Level Cascade (Multi-hop)
*Read this in [Russian (Русский)](#русский-russian).*

This fork of Hysteria 2 introduces a powerful new feature: **Protocol-level Cascading (Multi-hop)**.
Unlike traditional routing where chaining requires combining multiple proxy tools (e.g., shadowsocks -> vmess), this feature allows you to build a proxy chain consisting entirely of Hysteria 2 nodes (or compatible nodes like Xray-core) directly at the core protocol level.

## Features
* **Zero-overhead native routing:** The middle node runs an embedded instance of the official Hysteria 2 Client core. Traffic is seamlessly routed from the inbound QUIC stream directly into the outbound QUIC stream without leaving the application memory.
* **Client-driven topology:** The entire chain topology is defined and orchestrated by the PC client. Middle nodes do not need any special static routing configurations.
* **Per-hop Congestion Control (Brutal/BBR):** You can define specific upload and download speed limits for *each individual hop* directly in the client configuration. The middle nodes will automatically negotiate and apply the correct congestion control algorithms (Brutal or BBR) on a per-hop basis.
* **Xray-core Compatibility:** Note that Hysteria 2 masks datagram frame sizes by default for stealth. This causes a race condition with older `quic-go` versions (like v0.59.1 used in stock Xray-core). To cascade to an Xray-core endpoint, you **must** use a custom build of Xray-core updated to a newer `quic-go` library.

## How It Works
1. The user defines an array of `cascade` nodes in their local `client.yaml`.
2. The client connects to the first middle node and passes the remaining `cascade` array as a JSON string via the custom `Hysteria-Cascade` HTTP header during the initial QUIC authentication handshake.
3. The middle node unpacks the JSON, reads the next node's credentials and requested bandwidth, and dynamically initializes a native Hysteria 2 Client.
4. The middle node connects to the next node and bridges the `Outbound` interface directly into the new QUIC connection.
5. This process repeats until the endpoint node is reached.

## Configuration Example
You only need to configure the cascade on your local client PC. The middle nodes and exit nodes just need to be running standard (or compatible) Hysteria 2 server configurations.

**`client.yaml`**
```yaml
server: middle-node.com:443
auth: middle_secret
bandwidth:
  up: 50 mbps
  down: 50 mbps

cascade:
  - server: exit-node.com:443
    auth: exit_secret
    tls:
      sni: exit.com
      insecure: false
    bandwidth:
      up: 100 mbps
      down: 100 mbps
```

### Cascade Node Configuration Fields
* `server`: The address and port of the next node (e.g., `exit-node.com:443`).
* `auth`: The authentication password for the next node.
* `tls.sni`: (Optional) Server Name Indication for TLS. If omitted, it defaults to the hostname specified in `server`.
* `tls.insecure`: (Optional) Set to `true` to skip TLS certificate verification (useful for self-signed certificates).
* `bandwidth.up` / `bandwidth.down`: (Optional) Requested upload and download speeds for this specific hop.

## Bandwidth Negotiation & Congestion Control
To maintain Hysteria 2's high performance, congestion control is negotiated hop-by-hop rather than end-to-end:
1. **Hop 1 (Client ↔ Middle Node):** The speeds are negotiated based on the client's `bandwidth` (50 mbps) clamped by the Middle Node's global `server.yaml` limits.
2. **Hop 2 (Middle Node ↔ Exit Node):** The Middle Node calculates its *effective limits* by taking the minimum of the client's requested cascade bandwidth (`up: 100 mbps`, `down: 100 mbps`) and its own physical `server.yaml` limits. It then negotiates with the Exit Node using these effective limits.
3. **Algorithm Selection:** If speeds are defined and negotiated successfully, **Brutal** is used for that specific hop. If bandwidth blocks are omitted (0), the hop automatically falls back to **BBR**.

---

# Русский (Russian)

Данный форк Hysteria 2 представляет мощную новую функцию: **Каскадирование на уровне протокола (Multi-hop)**.
В отличие от классической маршрутизации, где для создания цепочки нужно комбинировать разные инструменты (например, shadowsocks -> vmess), эта функция позволяет строить прокси-цепочки, состоящие целиком из серверов Hysteria 2 (или совместимых с ними, например Xray-core), прямо на уровне самого протокола.

## Особенности
* **Нативная маршрутизация без накладных расходов:** Мидл-нода запускает внутри себя встроенный экземпляр официального ядра Hysteria 2 Client. Трафик бесшовно перекладывается из входящего QUIC-потока в исходящий QUIC-поток без выхода за пределы памяти приложения.
* **Топология, управляемая клиентом:** Вся цепочка узлов описывается и управляется исключительно с вашего ПК. На промежуточных серверах (мидл-нодах) не нужно прописывать никаких статических маршрутов.
* **Независимый контроль перегрузки (Brutal/BBR) для каждого участка:** Вы можете задавать индивидуальные лимиты скорости (upload/download) для *каждого отдельного участка (hop)* прямо в конфиге клиента. Мидл-ноды автоматически вычислят эффективные лимиты и применят нужный алгоритм (Brutal или BBR) независимо для каждого звена.
* **Совместимость с Xray-core:** Обратите внимание, что Hysteria 2 по умолчанию маскирует размер фрейма датаграмм для максимальной скрытности. Это вызывает гонку данных (race condition) при работе со старыми версиями `quic-go` (например, v0.59.1, которая используется в официальном Xray-core). Для каскадирования на Xray-core эндпоинт **необходимо** использовать кастомную сборку Xray-core, обновленную до свежей версии библиотеки `quic-go`.

## Как это работает
1. Пользователь задает массив `cascade` в своем локальном файле `client.yaml`.
2. Клиент подключается к первой мидл-ноде и во время первичного рукопожатия передает оставшийся массив `cascade` в виде JSON-строки через специальный HTTP-заголовок `Hysteria-Cascade`.
3. Мидл-нода распаковывает JSON, считывает данные следующей ноды (IP, пароль, запрошенные скорости) и динамически инициализирует нативного клиента Hysteria 2.
4. Мидл-нода подключается к следующему узлу и напрямую связывает интерфейс `Outbound` с новым QUIC-соединением.
5. Процесс повторяется вплоть до достижения конечной точки (эндпоинта).

## Пример конфигурации
Каскад настраивается только на локальном клиенте (ПК). Мидл-ноды и эндпоинты работают на обычных серверных конфигах Hysteria 2.

**`client.yaml`**
```yaml
server: middle-node.com:443
auth: middle_secret
bandwidth:
  up: 50 mbps
  down: 50 mbps

cascade:
  - server: exit-node.com:443
    auth: exit_secret
    tls:
      sni: exit.com
      insecure: false
    bandwidth:
      up: 100 mbps
      down: 100 mbps
```

### Параметры конфигурации каскадного узла
* `server`: Адрес и порт следующего узла (например, `exit-node.com:443`).
* `auth`: Пароль для авторизации на следующем узле.
* `tls.sni`: (Опционально) Server Name Indication для TLS. Если не указано, берется имя хоста из поля `server`.
* `tls.insecure`: (Опционально) Установите `true`, чтобы пропустить проверку TLS-сертификата (полезно для самоподписанных сертификатов).
* `bandwidth.up` / `bandwidth.down`: (Опционально) Запрошенные скорости загрузки и отдачи для конкретно этого участка сети.

## Согласование скоростей и контроль перегрузки (Congestion Control)
Для сохранения максимальной производительности Hysteria 2, алгоритмы перегрузки работают не сквозным образом, а индивидуально для каждого участка сети (hop-by-hop):
1. **Участок 1 (Клиент ↔ Мидл-нода):** Скорости согласовываются на основе главного блока `bandwidth` клиента (50 mbps), но не могут превысить глобальные лимиты `server.yaml` мидл-ноды.
2. **Участок 2 (Мидл-нода ↔ Эндпоинт):** Мидл-нода вычисляет свои *эффективные лимиты*: берет запрошенные в клиенте скорости каскада (`up: 100 mbps`, `down: 100 mbps`) и сравнивает их со своими физическими лимитами из `server.yaml`, выбирая минимум. Затем она использует эти эффективные лимиты для подключения к Эндпоинту.
3. **Выбор алгоритма:** Если лимиты скорости заданы (больше 0), на конкретном участке включается алгоритм **Brutal**. Если блоки `bandwidth` не указаны, данный участок автоматически переходит на использование динамического алгоритма **BBR**.
