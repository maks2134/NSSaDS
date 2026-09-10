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

## Сборка

```bash
cd lab8
make deps
make test
make build        # текущая платформа, CGO + OpenMPI
make build-all    # linux/darwin/windows amd64+arm64
```

Бинарники:

- `bin/lab8` — нативная сборка с MPI (`make build`)
- `bin/lab8-<os>-<arch>[.exe]` — кросс-сборка (`make build-all`)

Для сдачи на Linux-машинах собирайте **на каждой целевой ОС** с установленным OpenMPI (`make build`).

## Запуск

Локально (отладка):

```bash
make run-compare NP=4 GROUPS=2 N=512
# или:
mpirun -np 4 ./bin/lab8 -mode compare -groups 2 -seed 1 -n 512 -panel 64 -gen
```

На нескольких компьютерах:

1. Скопируйте `bin/lab8` на все хосты (одинаковый путь).
2. Положите `data/A.bin` и `data/B.bin` на **общий сетевой каталог** (или `-gen` с NFS `outdir`).
3. Заполните `hostfile` по образцу [`hostfile.example`](hostfile.example).
4. Запустите:

```bash
mpirun -np 6 --hostfile hostfile ./bin/lab8 \
  -mode compare -groups 2 -seed 1 -n 2560 -panel 64 -gen \
  -a /nfs/lab8/A.bin -b /nfs/lab8/B.bin -outdir /nfs/lab8
```

### Параметры

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `-n` | 2560 | размер матрицы `N` |
| `-panel` | 64 | ширина панели `B` для P2P-режимов |
| `-mode` | `compare` | `blocking` \| `nonblocking` \| `collective` \| `compare` |
| `-groups` | 2 | число групп |
| `-seed` | 1 | seed для случайного разбиения |
| `-a` / `-b` | `data/A.bin` / `data/B.bin` | входные матрицы |
| `-outdir` | `data` | каталог для `group-*.bin` |
| `-gen` | false | сгенерировать A/B на rank 0 перед запуском |

Пример вывода (rank 0):

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

1. Собрать на всех машинах, общий NFS для `data/`.
2. `mpirun --hostfile … -np ≥4 ./bin/lab8 -mode compare -groups 2 -gen -n …`
3. Показать разные составы групп и файлы `group-0.bin`, `group-1.bin`.
4. Сравнить `collective` vs `blocking` (и при желании `-mode nonblocking`).
5. Убедиться, что `checksum` и `checksum_p2p` совпадают внутри группы.

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
