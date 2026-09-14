# Лабораторная работа №7: MPI — парные коммуникации (blocking / non-blocking)

Умножение больших плотных матриц `N×N` (`float64`) через **point-to-point** MPI на Go (тонкая cgo-обёртка над OpenMPI). Два режима передачи панелей матрицы `B` по кольцу процессов:

1. **blocking** — `MPI_Send` / `MPI_Recv`, затем локальный GEMM (строго последовательно).
2. **nonblocking** — `MPI_Isend` / `MPI_Irecv` + двойной буфер панелей; пока идёт передача, выполняется `PanelGEMM` (аналог CUDA Streams + `cudaMemcpyAsync`).

Коллективные операции для данных не используются (только `MPI_Barrier` для синхронизации замера времени).

## Требования

- Go 1.26+ (CGO)
- OpenMPI (заголовки + runtime)
  - macOS: `brew install open-mpi`
  - Debian/Ubuntu: `sudo apt install libopenmpi-dev openmpi-bin`
- Для сдачи: запуск минимум на **3 физических машинах** (`mpirun --hostfile …`)

## Сборка

```bash
cd lab7
make deps
make test
make build        # текущая платформа, CGO + OpenMPI
make build-all    # linux/darwin/windows amd64+arm64
```

Бинарники:

- `bin/lab7` — нативная сборка с MPI (`make build`)
- `bin/lab7-<os>-<arch>[.exe]` — кросс-сборка (`make build-all`)

Для сдачи на Linux-машинах собирайте **на каждой целевой ОС** с установленным OpenMPI (`make build`). Кросс-бинарники без CGO стартуют с сообщением, что MPI недоступен (нужен OpenMPI + CGO на Unix).

## Запуск (только make)

Всё показывается через `make`. Параметры по умолчанию: `N=2560`, `panel=64`, `np=3`.

| Команда | Когда |
|---------|--------|
| `make demo` | Репетиция на **1 ПК** (оба режима подряд) |
| `make defend` | **Сдача на 3 ПК** (оба режима через hostfile) |
| `make blocking` / `make nonblocking` | Один режим локально |
| `make blocking-cluster` / `make nonblocking-cluster` | Один режим на кластере |
| `make questions` | Ответы на контрольные вопросы |
| `make help` | Список целей |

Если прогон слишком короткий (~10–50 с нужно):

```bash
make demo N=3072
make defend N=3072
```

На rank 0 печатается: режим, `P`, `N`, panel, wall-time (`MPI_Wtime`), checksum `C` (должен совпадать в обоих режимах).
## Алгоритм

1. Rank 0 генерирует `A` и `B`, раздаёт строки `A` парными `Send`/`Recv`.
2. Панели столбцов `B` идут по кольцу: `0 → 1 → … → P-1`.
3. Каждый ранг накапливает свой блок строк `C_local += A_local * B_panel`.
4. Блоки `C` собираются на rank 0 для checksum.

**Non-blocking:** двойной буфер панелей. Пока `Irecv` следующей панели / `Isend` текущей в следующий ранг, идёт умножение текущей панели. `Wait` только когда буфер нужен снова.

## Сценарий сдачи

**Сколько машин:** **минимум 3 физических ПК**. Два — мало. Один ПК — только репетиция (`make demo`).

### Подготовка (на каждом из 3 ПК)

```bash
# Go + OpenMPI, затем:
cd lab7
make build          # одинаковый путь к bin/lab7 на всех хостах
```

SSH без пароля с машины-запускателя на два остальных. OpenMPI стартует процессы по SSH.

### Показ преподавателю (make)

```bash
# 1) один раз записать три хоста
make hostfile HOSTS=192.168.1.10,192.168.1.11,192.168.1.12

# 2) оба режима подряд (blocking → nonblocking)
make defend

# 3) ответы на вопросы
make questions
```

Репетиция дома на одном ПК:

```bash
make demo
```

Сравнить на экране: `P=3`, **одинаковый checksum**, **nonblocking быстрее**. Если время < ~10 с — `make defend N=3072`.

### Чеклист

| Что | Ожидание |
|-----|----------|
| ПК | ≥ 3 |
| Команда | `make defend` |
| Checksum | совпадает |
| Время | nonblocking быстрее |
| Вопросы | `make questions` |

## Архитектура

```
lab7/
├── cmd/lab7/                      # CLI entry
├── internal/domain/               # Matrix, PanelGEMM, Transport interface
├── internal/infrastructure/mpi/   # cgo: Init/Finalize, Send/Recv, Isend/Irecv/Wait
├── internal/usecase/              # blocking + nonblocking multiply
├── pkg/config/
├── hostfile.example
└── Makefile
```

## Контрольные вопросы

### 1. Что такое MPI_COMM_WORLD?

`MPI_COMM_WORLD` — предопределённый **коммуникатор**, включающий все процессы MPI-приложения, запущенные данным `mpirun`. Через него задаётся «пространство» рангов и выполняются обмены по умолчанию. (В задании опечатка: `MPI_Com_World`.)

### 2. Что такое rank?

**Rank** — целочисленный идентификатор процесса внутри коммуникатора, от `0` до `size-1`. Его возвращает `MPI_Comm_rank`. По rank выбирают роль (например, root = 0) и адресата парных операций.

### 3. Какими вызовами должна начинаться и завершаться MPI-программа?

- Начало: `MPI_Init` (или `MPI_Init_thread`).
- Конец: `MPI_Finalize`.

В этой лабе: `mpi.Start` → `MPI_Init`, `session.Stop` → `MPI_Finalize`. Между ними допустимы только вызовы MPI API.

### 4. Объяснить преимущество асинхронных операций

Неблокирующие `Isend`/`Irecv` **инициируют** передачу и сразу возвращают управление. Пока сеть копирует данные, процесс может считать (`PanelGEMM`). Это **перекрытие коммуникации и вычислений** (как async memcpy + kernel в CUDA streams): уменьшается простой CPU в ожидании сети, особенно на нескольких физических машинах.
