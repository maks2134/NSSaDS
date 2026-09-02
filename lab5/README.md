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

Демонстрация подмены адреса источника в IP-заголовке. Отправляется не больше **5** пакетов (по умолчанию 3). Это не flood.

```bash
sudo ./bin/lab5 smurf -victim 192.168.1.10 -bcast 192.168.1.255 -c 3
```

На атакуемом узле в Wireshark:

```
icmp && ip.dst == 192.168.1.10
```

Ожидается ICMP Echo Reply от узлов сегмента, потому что в запросах source = адрес жертвы, dest = broadcast.

Запускать только на своей лабораторной сети, с согласия участников. Directed broadcast на многих сетях выключен — для сдачи нужен сегмент, где ICMP на broadcast разрешён (или явный directed broadcast).

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
