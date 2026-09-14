# Лабораторная работа №5: параллельный ping, traceroute, учебный Smurf

Кроссплатформенная ICMP-утилита на Go. Один процесс открывает raw ICMP-сокет, каждый хост обрабатывается отдельной горутиной. Чтобы потоки не забирали чужие ответы, пакет сначала читается с `MSG_PEEK` и удаляется из буфера только после проверки ICMP Identifier/Sequence.

## Требования

- Go 1.26+
- root (Linux/macOS) или Administrator (Windows) — raw-сокеты
- IPv4; сдача на 2+ физических машинах

## Сборка

```bash
make deps
make test
make build
# кросс-сборка Windows/Linux/macOS:
make build-all
```

## Ping нескольких хостов

```bash
sudo ./bin/lab5 ping 8.8.8.8 1.1.1.1
sudo ./bin/lab5 ping -c 4 -W 1s --trace 192.168.1.1 8.8.8.8
```

- `-c N` — число запросов (по умолчанию 4)
- `-W duration` — таймаут ответа
- `-i interval` — интервал между запросами
- `--trace` — traceroute до каждого хоста, затем ping

Метка времени отправки кладётся в тело echo-запроса (8 байт unix nano). RTT считается по этой метке в ответе.

Обработчики ICMP разделены:

- Echo Reply
- Time Exceeded (TTL истёк)
- Destination Unreachable (хост недостижим)

## Traceroute

```bash
sudo ./bin/lab5 traceroute -m 30 8.8.8.8
sudo ./bin/lab5 traceroute 192.168.1.1 8.8.8.8
```

TTL увеличивается с 1. На Time Exceeded печатается промежуточный узел, на Echo Reply путь считается найденным, на Host Unreachable — остановка с диагностикой.

## Учебный Smurf (только своя лабораторная сеть)

Демонстрация подмены адреса источника в IP-заголовке. Не больше **5** пакетов. Для двух ПК достаточно IP жертвы — broadcast и «отражатель» (этот компьютер) подставляются сами.

```bash
# на атакующем ПК, IP — адрес жертвы
sudo ./bin/lab5 smurf 192.168.1.10
```

На атакуемом узле в Wireshark:

```
icmp && ip.dst == 192.168.1.10
```

На жертве в Wireshark: `icmp && ip.dst == <IP-жертвы>`. Должны прийти Echo Reply на её адрес (атакующий ПК отвечает на поддельный echo). Запускать только на своей лабораторной сети.

## Проверка на двух компьютерах

Нужны две машины в **одной L2-сети** (один Wi‑Fi / один Ethernet-свитч, без гостевой изоляции клиентов). Ниже: **ПК-A** — запускает `lab5`, **ПК-B** — цель ping/traceroute и «жертва» для Smurf (на нём Wireshark).

### 0. Подготовка

На обеих машинах:

1. Соберите бинарник (`make build`) или скопируйте `bin/lab5` с первой машины.
2. Отключите блокировку ICMP в файрволе (иначе ping «молчит»).
3. Узнайте адреса:

```bash
# Linux
ip -4 addr show
ip -4 route show | grep default

# macOS
ifconfig | grep "inet "

# Windows (PowerShell)
ipconfig
```

Пример: ПК-A = `192.168.1.20`, ПК-B = `192.168.1.10`, маска `/24` → broadcast `192.168.1.255`.

Проверка, что машины видят друг друга (системный ping):

```bash
# на ПК-A
ping -c 3 192.168.1.10
```

Если системный ping не проходит — сначала сеть/файрвол, `lab5` тоже не заработает.

Linux/macOS запускайте через `sudo`. Windows — «Запуск от имени администратора».

### Если не идёт даже обычный `ping`

Сначала в Terminal, **без** lab5:

```bash
ping -c 3 127.0.0.1      # должно ответить
ping -c 3 8.8.8.8        # интернет
ping -c 3 <IP-второго-ПК>
```

- `127.0.0.1` ок, а второй ПК нет — вы **не в одной сети** или указан чужой адрес из примера (`192.168.1.10` подставлять нельзя).
- Смотрите **реальный** IPv4: `ifconfig` / `ip addr`. Сейчас типично iPhone-модем `172.20.10.x` или VPN `utun*` / `10.x`. Пинговать надо адрес, который показывает **вторая** машина, из той же подсети.
- Default через `utun` (VPN/корпоративный туннель) часто режет ICMP до соседа по Wi‑Fi. На время сдачи выключите VPN.
- Личный хотспот / «гостевой» Wi‑Fi с Client Isolation: клиенты не пингуются друг с другом. Нужен обычный роутер/Ethernet.
- macOS Firewall может резать входящий ICMP на ПК-B.

lab5 на macOS использует ICMP datagram (как системный ping): raw-сокет там не получает Echo Reply. Если системный ping до хоста проходит, а lab5 нет — пришлите вывод `sudo ./bin/lab5 ping -c 2 <IP>`.

### 1. Параллельный ping (ПК-A → ПК-B и ещё один хост)

На **ПК-B** откройте Wireshark на том же интерфейсе, что в LAN, фильтр:

```
icmp && ip.addr == 192.168.1.10
```

На **ПК-A**:

```bash
sudo ./bin/lab5 ping -c 4 192.168.1.10 8.8.8.8
```

Ожидается:

- две горутины сразу: ответы и от ПК-B, и от `8.8.8.8` (если есть интернет);
- строки вида `bytes from 192.168.1.10 ... icmp_seq=... time=... ms`;
- в Wireshark на ПК-B: Echo Request с ПК-A и Echo Reply обратно.

Без второго внешнего хоста достаточно двух адресов в LAN (ПК-B и шлюз):

```bash
sudo ./bin/lab5 ping -c 4 192.168.1.10 192.168.1.1
```

### 2. Traceroute до ПК-B

На **ПК-A**:

```bash
sudo ./bin/lab5 traceroute 192.168.1.10
```

В одной подсети обычно один hop — сразу Echo Reply с ПК-B. Чтобы увидеть Time Exceeded, трассируйте узел за шлюзом:

```bash
sudo ./bin/lab5 traceroute -m 15 8.8.8.8
sudo ./bin/lab5 ping --trace -c 2 192.168.1.10 8.8.8.8
```

В Wireshark на ПК-B при traceroute до него: ICMP Echo Request с TTL=1,2,… пока пакет не дойдёт.

### 3. Учебный Smurf: 2 ПК → 1 жертва

Нужны **две машины в одной LAN** (VPN выключить). Третий хост не нужен: атакующий ПК сам отвечает на spoofed echo, плюс уходит пакет на broadcast.

1. На **жертве (ПК-B)** узнай IPv4 (`ifconfig` / `ipconfig`) и запусти Wireshark **до** атаки, фильтр:

```
icmp && ip.dst == <IP-жертвы>
```

2. На **атакующем (ПК-A)** — не на жертве:

```bash
make build
sudo ./bin/lab5 smurf <IP-жертвы>
```

Если атакующих два (оба бьют в одну жертву) — та же команда на каждом из них.

Опционально: `-c 3`, `-bcast 192.168.1.255` если авто-broadcast не тот.

В Wireshark на жертве: Echo Reply с атакующего (src = ПК-A, dst = жертва), хотя жертва никого не пинговала. Если broadcast живой — ещё Echo Request с поддельным source = жертва.

### 4. Файрвол (если ping не отвечает)

```bash
# Linux (пример nft/ufw)
sudo ufw allow proto icmp
# или
sudo iptables -I INPUT -p icmp -j ACCEPT

# macOS: Системные настройки → Сеть → Фаервол → разрешить ICMP / отключить на время сдачи

# Windows: разрешить ICMPv4-In в брандмауэре Защитника Windows
netsh advfirewall firewall add rule name="ICMPv4" protocol=icmpv4:8,any dir=in action=allow
```

После сдачи правило лучше удалить.

## Архитектура

```
lab5/
├── cmd/lab5/                 # CLI: ping / traceroute / smurf
├── internal/domain/       # сущности и интерфейсы сокета
├── internal/infrastructure/icmp/
│   ├── socket_unix.go      # Send/Peek/Recv/SetTTL
│   ├── socket_windows.go
│   ├── rawip_unix.go       # IP_HDRINCL для Smurf
│   ├── rawip_windows.go
│   └── peek.go             # MSG_PEEK + mutex
└── internal/usecase/       # ping, traceroute, smurf
```

Платформенные вызовы спрятаны внутри `Open` / `Send` / `Peek` / `Recv`. Бизнес-логика общая для Windows и Unix.

## Замечания по реализации

- Чтение пакетами до 64 KiB, не по одному байту
- Границы ICMP-сообщения выделяются разбором заголовка, а не одним «удачным» `recv`
- Общий сокет: `Peek` → проверка принадлежности воркеру → `Recv` без `MSG_PEEK`
- Завершение по Ctrl+C (`SIGINT`), без самопроизвольного выхода серверной части (утилита клиентская)
