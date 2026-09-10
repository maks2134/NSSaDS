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

## Запуск

Локально (3 процесса на одной машине — для отладки):

```bash
make run-blocking
make run-nonblocking
# или:
mpirun -np 3 ./bin/lab7 -mode blocking -n 2560 -panel 64
mpirun -np 3 ./bin/lab7 -mode nonblocking -n 2560 -panel 64
```

На 3 компьютерах:

1. Скопируйте `bin/lab7` на все хосты (одинаковый путь).
2. Заполните `hostfile` по образцу [`hostfile.example`](hostfile.example).
3. Запустите:

```bash
mpirun -np 3 --hostfile hostfile ./bin/lab7 -mode blocking -n 2560 -panel 64
mpirun -np 3 --hostfile hostfile ./bin/lab7 -mode nonblocking -n 2560 -panel 64
```

Параметры:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `-n` | 2560 | размер матрицы `N` (подберите так, чтобы прогон занимал ~10–50 с) |
| `-panel` | 64 | ширина панели `B` в столбцах |
| `-mode` | `blocking` | `blocking` или `nonblocking` |

На rank 0 печатается: режим, число процессов `P`, `N`, panel, wall-time (`MPI_Wtime`), checksum `C` (должен совпадать в обоих режимах).

## Алгоритм

1. Rank 0 генерирует `A` и `B`, раздаёт строки `A` парными `Send`/`Recv`.
2. Панели столбцов `B` идут по кольцу: `0 → 1 → … → P-1`.
3. Каждый ранг накапливает свой блок строк `C_local += A_local * B_panel`.
4. Блоки `C` собираются на rank 0 для checksum.

**Non-blocking:** двойной буфер панелей. Пока `Irecv` следующей панели / `Isend` текущей в следующий ранг, идёт умножение текущей панели. `Wait` только когда буфер нужен снова.

## Сценарий сдачи

1. Собрать на всех машинах (или NFS), подготовить `hostfile` с ≥3 хостами.
2. Запустить blocking и nonblocking с одинаковыми `-n` / `-panel`.
3. Сравнить времена: nonblocking должен быть заметно быстрее на реальной сети.
4. Убедиться, что `checksum` совпадает.
5. Ответить на контрольные вопросы (ниже).

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
