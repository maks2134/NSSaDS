# Лабораторная работа №8: MPI — групповые и файловые операции

Расширение [л.р. №7](../lab7): коллективные операции, случайные группы процессов и параллельный MPI-IO.

Программа:

1. Разбивает `MPI_COMM_WORLD` на **K групп** (`-groups`, случайный состав, seed задаётся).
2. Каждая группа **независимо** умножает матрицы `N×N`.
3. Исходные `A`/`B` читаются из общих файлов (NFS) коллективным MPI-IO по rank.
4. Результат каждой группы пишется в `group-{color}.bin`.
5. Замеряет время **collective** (`MPI_Bcast`) и **P2P** (ring `Send`/`Recv` из л.р. №7) и печатает сравнение.

## Требования

- Go 1.26+ (CGO)
- OpenMPI (заголовки + runtime + MPI-IO)
  - macOS: `brew install open-mpi`
  - Debian/Ubuntu: `sudo apt install libopenmpi-dev openmpi-bin`
- Для сдачи: запуск на **2+ физических машинах** (`mpirun --hostfile …`), общий путь к `data/`

## Показ лабораторной (только `make`)

Все сценарии сдачи завёрнуты в Makefile — флаги MPI/CLI передавать не нужно.

```bash
cd lab8
make help                 # список целей
make demo-quick           # быстрый полный прогон (N=256): тесты + compare + nonblocking + файлы
make demo                 # то же с дефолтами N=2560 NP=4 GROUPS=2
make run                  # основной прогон: compare (коллективы vs P2P) + группы + MPI-IO
make show                 # показать A.bin / B.bin / group-*.bin
```

Отдельные режимы:

```bash
make run-collective
make run-blocking
make run-nonblocking
```

Переопределять только то, что меняете:

```bash
make run NP=6 GROUPS=3 N=1024
make demo-quick GROUPS=3
```

| Переменная | По умолчанию | Смысл |
|------------|--------------|--------|
| `NP` | 4 | число MPI-процессов |
| `GROUPS` | 2 | число групп |
| `N` | 2560 | размер матрицы |
| `PANEL` | 64 | ширина панели B (P2P) |
| `SEED` | 1 | seed разбиения на группы |
| `HOSTFILE` | пусто | путь к hostfile для кластера |

### Кластер (2+ машины)

1. `make build` на каждой машине (или общий бинарник).
2. Скопируйте [`hostfile.example`](hostfile.example) → `hostfile`, пропишите хосты.
3. Общий каталог `data/` (NFS) либо локальный `data/` с `-gen` на rank 0.

```bash
make run HOSTFILE=hostfile NP=6
make demo HOSTFILE=hostfile NP=6 N=1024
```

### Сборка

```bash
make deps && make test && make build
make build-all    # кросс linux/darwin/windows (реальный MPI только нативный Unix + CGO)
```

Пример вывода (`make run`):

```
groups=2  N=512  panel=64  seed=1  mode=compare
group=0  ranks=[0 2 3]  P=3  collective=0.412s  blocking=0.501s  checksum=1.234567e+06
group=1  ranks=[1]  P=1  collective=1.103s  blocking=1.098s  checksum=1.234567e+06
```
## Алгоритм

### Группы

1. Rank 0 строит цвета `0..K-1` (`AssignColors`: при `K≤P` — K непустых случайных групп; при `K>P` — синглтоны).
2. `MPI_Bcast` массива цветов.
3. `MPI_Comm_split(color, key=worldRank)`.

### Файлы

Формат: `int64 N` (LE) + `N×N float64` row-major.

- Каждый процесс группы читает **полосу строк A** (`MPI_File_read_at_all`) и **столбцы B** (`MPI_Type_create_subarray` + `MPI_File_read_all`).
- После умножения пишет свои строки **C** в `group-{color}.bin` (`MPI_File_write_at_all`).

### Умножение

- **Collective:** владелец столбцов `B` делает `MPI_Bcast` своей панели → `PanelGEMM`.
- **P2P (blocking/nonblocking):** `Gatherv` собирает полный `B` на group rank 0, затем кольцо панелей как в л.р. №7.
- Режим `compare` гоняет оба алгоритма на одних и тех же локальных данных и печатает оба времени.

## Сценарий сдачи

```bash
make build
make demo-quick                          # локально / репетиция
make run HOSTFILE=hostfile NP=6          # на 2+ машинах
make show                                # group-0.bin, group-1.bin, …
```

1. Собрать на всех машинах (`make build`), общий NFS для `data/` при необходимости.
2. `make run` / `make demo` — compare: время collective vs blocking, разные `ranks=` у групп.
3. `make show` — файлы результатов групп.
4. При необходимости `make run-nonblocking` — асинхронный режим из л.р. №7.
5. `checksum` и `checksum_p2p` внутри группы должны совпадать.

## Архитектура

```
lab8/
├── cmd/lab8/                         # CLI
├── internal/domain/                  # Matrix, AssignColors, file format
├── internal/infrastructure/mpi/      # cgo: Comm, Split, Bcast, Gatherv, MPI-IO
├── internal/usecase/                 # groups, IO, collective/blocking/nonblocking
├── pkg/config/
├── hostfile.example
└── Makefile
```

## Контрольные вопросы

### 1. Что делает `MPI_Comm_split`?

Создаёт **новые коммуникаторы** из процессов с одинаковым `color`. Процессы с разными цветами попадают в разные группы; внутри группы ранги упорядочиваются по `key`. Пустой color не создаёт коммуникатор.

### 2. Чем `MPI_Bcast` отличается от парных `Send`/`Recv`?

`Bcast` — **коллектив**: все процессы коммуникатора участвуют одним вызовом, root рассылает одинаковый буфер всем. Парные операции адресуют одного получателя/отправителя и требуют явного цикла рассылки.

### 3. Зачем MPI-IO вместо обычного `open`/`read`?

Коллективный MPI-IO (`read_at_all`, `read_all` + file view) координирует доступ процессов к общему файлу, позволяет задать **view** (subarray) и читать только свою порцию без гонок и лишних POSIX-seek на каждом ранге. На параллельных ФС это эффективнее независимого POSIX I/O.

### 4. Что такое file view / subarray?

`MPI_File_set_view` задаёт, какую часть файла «видит» процесс. `MPI_Type_create_subarray` описывает прямоугольный блок в массиве (например, столбцы `B`). После установки view коллективный `MPI_File_read_all` читает только этот блок в плотный локальный буфер.
