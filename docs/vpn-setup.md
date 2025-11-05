# Подробное руководство по настройке VPN для проекта

## Оглавление

1. [Введение](#введение)
2. [Как это работает](#как-это-работает)
3. [Пошаговая настройка](#пошаговая-настройка)
4. [Использование](#использование)
5. [Troubleshooting](#troubleshooting)
6. [FAQ](#faq)

## Введение

Это руководство описывает, как настроить проект для работы через VPN (Amnezia) таким образом, чтобы:

1. WSL имел доступ к интернету при подключенном VPN на Windows
2. Ваши друзья, подключенные к тому же VPN, могли использовать ваш локальный сервис

### Целевая архитектура

```
┌────────────────────────────────────────┐
│          Windows с Amnezia VPN         │
│          VPN IP: 10.8.0.5              │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │           WSL2                   │ │
│  │  ┌────────────────────────────┐  │ │
│  │  │   Docker Containers        │  │ │
│  │  │  - REST API :8080          │  │ │
│  │  │  - auth-service :50051     │  │ │
│  │  │  - PostgreSQL :5432        │  │ │
│  │  └────────────────────────────┘  │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘
                 ▲
                 │ VPN (например, 10.8.0.0/24)
                 │
    ┌────────────┴───────────────┐
    │                            │
┌───┴────┐                  ┌────┴────┐
│ Друг 1 │                  │ Друг 2  │
│10.8.0.6│                  │10.8.0.7 │
└────────┘                  └─────────┘
```

## Как это работает

### Проблема с DNS в WSL

По умолчанию WSL2 автоматически генерирует файл `/etc/resolv.conf`, получая DNS серверы от Windows. Когда вы подключаетесь к VPN на Windows:

1. VPN клиент изменяет DNS настройки Windows
2. WSL пытается использовать DNS сервер VPN
3. VPN DNS сервер может не отвечать из WSL или давать неправильные результаты
4. Результат: интернет в WSL перестает работать

### Решение

Мы отключаем автогенерацию `/etc/resolv.conf` и настраиваем статические публичные DNS серверы (Google, Cloudflare), которые всегда доступны.

### Проброс портов

Docker в WSL2 автоматически пробрасывает порты в Windows через механизм WSL networking:

1. Docker контейнер слушает на `0.0.0.0:8080` внутри WSL
2. WSL пробрасывает порт в Windows (доступен как `localhost:8080` в Windows)
3. Windows делает порт доступным на всех сетевых интерфейсах, включая VPN
4. Друзья по VPN могут подключаться к `ВАШ_VPN_IP:8080`

## Пошаговая настройка

### Шаг 1: Обновление WSL (если требуется)

Mirrored networking требует WSL версии 2.0 или выше.

В **PowerShell** проверьте версию:

```powershell
wsl --version
```

Если версия ниже 2.0 или команда не работает, обновите WSL:

```powershell
# От администратора
wsl --update
```

### Шаг 2: Установка Amnezia VPN (если еще не установлен)

1. Скачайте клиент с [официального сайта](https://amnezia.org/)
2. Получите конфигурацию VPN от администратора VPN сервера
3. Подключитесь к VPN и убедитесь что интернет работает в Windows

### Шаг 3: Настройка WSL для работы с VPN (Mirrored Networking)

Это современное решение, которое делает WSL сеть "зеркальной" с Windows, автоматически решая все проблемы с VPN.

#### 3.1. Создание файла конфигурации WSL

В **PowerShell** (Windows) выполните:

```powershell
# Создайте файл .wslconfig в домашней директории
$wslConfig = @"
[wsl2]
networkingMode=mirrored
dnsTunneling=true
firewall=true
autoProxy=true
"@

$wslConfig | Out-File -FilePath "$env:USERPROFILE\.wslconfig" -Encoding ASCII

# Проверьте что файл создан
Get-Content "$env:USERPROFILE\.wslconfig"
```

**Что делают эти настройки:**

- `networkingMode=mirrored` - WSL использует тот же сетевой стек что и Windows. Это означает:
  - Одинаковые IP адреса для всех интерфейсов
  - Автоматическая работа с VPN
  - Нет проблем с маршрутизацией
  
- `dnsTunneling=true` - DNS запросы автоматически проксируются через Windows
  - WSL использует те же DNS что и Windows (включая VPN DNS)
  - Нет необходимости вручную настраивать `/etc/resolv.conf`
  
- `firewall=true` - применяет правила Windows Firewall к WSL трафику
  - Дополнительная безопасность
  - Единая политика безопасности для Windows и WSL
  
- `autoProxy=true` - автоматически использует proxy настройки Windows
  - Полезно для корпоративных сетей

#### 3.2. Перезапуск WSL

Закройте все окна WSL и выполните в PowerShell (Windows):

```powershell
wsl --shutdown
```

Подождите 10 секунд, затем откройте WSL снова.

#### 3.3. Проверка интернет-соединения

```bash
# Проверка DNS резолвинга
nslookup google.com

# Проверка пинга
ping -c 3 google.com

# Проверка HTTP запроса
curl -I https://google.com
```

Все команды должны работать успешно.

### Шаг 3: Проверка конфигурации Docker Compose

Убедитесь что порты в `deploy/docker-compose/docker-compose.yaml` биндятся на `0.0.0.0`:

```yaml
services:
  rest-api:
    ports:
      - "8080:8080"  # Автоматически биндится на 0.0.0.0
```

Docker Compose по умолчанию биндит на `0.0.0.0`, поэтому дополнительная настройка не требуется.

## Использование

### Запуск проекта

1. **Включите VPN на Windows:**
   - Откройте Amnezia клиент
   - Подключитесь к вашему VPN серверу

2. **Запустите проект в WSL:**

```bash
cd /home/main/projects/golang-project
make docker-up
```

3. **Проверьте что сервисы работают:**

```bash
# Внутри WSL
curl http://localhost:8080/health

# Должен вернуть статус OK
```

### Получение VPN IP адреса

#### Способ 1: PowerShell (Windows)

```powershell
ipconfig
```

Найдите адаптер с названием типа:
- "Amnezia"
- "WireGuard"  
- "OpenVPN"
- "Tunnel Adapter"

Пример вывода:

```
Ethernet adapter Amnezia:

   Connection-specific DNS Suffix  . :
   IPv4 Address. . . . . . . . . . . : 10.8.0.5
   Subnet Mask . . . . . . . . . . . : 255.255.255.0
   Default Gateway . . . . . . . . . : 10.8.0.1
```

Ваш VPN IP: `10.8.0.5`

#### Способ 2: Интерфейс Amnezia

В клиенте Amnezia обычно отображается ваш VPN IP адрес в статусе подключения.

#### Способ 3: Автоматически (PowerShell)

```powershell
# Для WireGuard
(Get-NetIPAddress -InterfaceAlias "*WireGuard*" -AddressFamily IPv4).IPAddress

# Для OpenVPN
(Get-NetIPAddress -InterfaceAlias "*TAP*" -AddressFamily IPv4).IPAddress
```

### Предоставление доступа друзьям

Передайте друзьям:
1. Ваш VPN IP адрес (например, `10.8.0.5`)
2. Информацию о доступных эндпоинтах

Пример сообщения:

```
Привет! Я запустил сервис. Подключайтесь:

Base URL: http://10.8.0.5:8080

Эндпоинты:
- GET  /health
- POST /api/v1/auth/signup
- POST /api/v1/auth/signin
- GET  /api/v1/auth/validate

Пример регистрации:
curl -X POST http://10.8.0.5:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123!"}'
```

### Проверка доступа со стороны друга

Друг должен:

1. Подключиться к тому же VPN серверу
2. Выполнить тестовый запрос:

```bash
curl http://10.8.0.5:8080/health
```

Если получен ответ - всё работает!

## Troubleshooting

### Проблема: Интернет не работает в WSL после настройки

**Симптомы:**
```bash
$ ping google.com
ping: google.com: Temporary failure in name resolution
```

**Решение:**

1. **Проверьте что используется Mirrored Networking:**

В PowerShell (Windows):
```powershell
Get-Content "$env:USERPROFILE\.wslconfig"
```

Должно содержать:
```ini
[wsl2]
networkingMode=mirrored
dnsTunneling=true
firewall=true
autoProxy=true
```

2. **Проверьте версию WSL:**

```powershell
wsl --version
```

Mirrored networking требует WSL 2.0+. Если версия старая, обновите:

```powershell
wsl --update
```

3. **Перезапустите WSL после создания .wslconfig:**

```powershell
wsl --shutdown
```

Подождите 10 секунд и откройте WSL снова.

4. **Проверьте что VPN подключен в Windows:**

Если VPN отключен, WSL тоже не будет иметь интернет через VPN.

5. **Если используете старую версию WSL (без mirrored networking):**
Используйте старый метод с ручной настройкой DNS:

```bash
# Отключите автогенерацию resolv.conf
sudo nano /etc/wsl.conf
# Добавьте:
# [network]
# generateResolvConf = false

# Перезапустите WSL (из PowerShell: wsl --shutdown)

# Настройте DNS вручную
sudo rm -f /etc/resolv.conf
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 1.1.1.1" | sudo tee -a /etc/resolv.conf
```

6. Попробуйте альтернативные DNS (если ручной способ):
```bash
# Yandex DNS
echo "nameserver 77.88.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 77.88.8.1" | sudo tee -a /etc/resolv.conf

# OpenDNS
echo "nameserver 208.67.222.222" | sudo tee /etc/resolv.conf
echo "nameserver 208.67.220.220" | sudo tee -a /etc/resolv.conf
```

### Проблема: Друзья не могут подключиться к сервису

**Симптомы:**
```bash
$ curl http://10.8.0.5:8080/health
curl: (7) Failed to connect to 10.8.0.5 port 8080: Connection refused
```

**Решение:**

1. **Проверьте что Docker контейнеры работают (на вашей стороне):**
```bash
cd /home/main/projects/golang-project/deploy/docker-compose
docker-compose ps
```

Все сервисы должны быть в состоянии `Up`.

2. **Проверьте доступность изнутри WSL:**
```bash
curl http://localhost:8080/health
```

Если работает, переходите к следующему шагу.

3. **Проверьте доступность из Windows:**

В PowerShell:
```powershell
curl http://localhost:8080/health
```

Если работает, переходите к следующему шагу.

4. **Проверьте Windows Firewall:**

В PowerShell (от администратора):
```powershell
# Добавьте правило для порта 8080
New-NetFirewallRule -DisplayName "Golang REST API" -Direction Inbound -LocalPort 8080 -Protocol TCP -Action Allow

# Проверьте существующие правила
Get-NetFirewallRule -DisplayName "Golang*"
```

5. **Проверьте что друг в той же VPN сети:**

Попросите друга выполнить:
```bash
# Проверка пинга
ping 10.8.0.5

# Если пинг не проходит - проблема с VPN подключением
```

6. **Проверьте настройки VPN сервера:**

Возможно, на VPN сервере настроена изоляция клиентов (client-to-client блокировка). Обратитесь к администратору VPN сервера.

### Проблема: Docker контейнеры не запускаются

**Симптомы:**
```bash
$ make docker-up
Error response from daemon: driver failed programming external connectivity...
```

**Решение:**

1. **Проверьте что порты не заняты:**
```bash
# Проверка портов
sudo netstat -tulpn | grep -E ':(8080|50051|5432)'

# Или
sudo lsof -i :8080
sudo lsof -i :50051
sudo lsof -i :5432
```

2. **Остановите конфликтующие сервисы:**
```bash
# Если есть старые контейнеры
make docker-down

# Полная очистка
make docker-clean
```

3. **Перезапустите Docker:**
```bash
sudo systemctl restart docker

# Или в WSL2
wsl --shutdown
# Откройте WSL снова
```

4. **Запустите заново:**
```bash
make docker-up
```

### Проблема: Медленное подключение через VPN

**Симптомы:**
- Запросы выполняются очень долго
- Таймауты при подключении

**Решение:**

1. **Проверьте пинг до вашего VPN IP:**
```bash
ping 10.8.0.5
```

Пинг должен быть < 100ms для комфортной работы.

2. **Проверьте нагрузку на VPN сервер:**
- Возможно сервер перегружен
- Попробуйте подключиться в другое время

3. **Оптимизируйте Docker логирование:**

В `deploy/docker-compose/docker-compose.yaml` добавьте:

```yaml
services:
  rest-api:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Проблема: После обновления Windows настройки сбросились

**Симптомы:**
После обновления Windows интернет в WSL снова перестал работать при подключенном VPN.

**Решение:**

Обновления Windows иногда могут сбросить настройки WSL. Повторите шаги 2.3 из раздела настройки:

```bash
# Проверьте resolv.conf
cat /etc/resolv.conf

# Если он снова автогенерируемый, пересоздайте
sudo rm /etc/resolv.conf
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 1.1.1.1" | sudo tee -a /etc/resolv.conf
```

## FAQ

### Q: Можно ли использовать свой DNS сервер вместо 8.8.8.8?

**A:** Да, можете использовать любой публичный DNS:

```bash
# Yandex DNS
nameserver 77.88.8.8
nameserver 77.88.8.1

# OpenDNS
nameserver 208.67.222.222
nameserver 208.67.220.220

# Quad9
nameserver 9.9.9.9
nameserver 149.112.112.112
```

### Q: Могу ли я добавить DNS сервер VPN в дополнение к публичным?

**A:** Да, можете добавить несколько DNS серверов:

```bash
echo "nameserver 10.8.0.1" | sudo tee /etc/resolv.conf      # VPN DNS
echo "nameserver 8.8.8.8" | sudo tee -a /etc/resolv.conf    # Google (fallback)
echo "nameserver 1.1.1.1" | sudo tee -a /etc/resolv.conf    # Cloudflare (fallback)
```

DNS будут использоваться по порядку.

### Q: Будет ли работать без VPN?

**A:** Да! Настройка WSL с публичными DNS работает и без VPN. Это просто позволяет WSL иметь стабильный интернет в любой ситуации.

### Q: Нужно ли открывать порты на роутере?

**A:** Нет! Всё работает внутри VPN сети, порты на роутере открывать не нужно. Это даже более безопасно.

### Q: Могу ли я использовать несколько VPN одновременно?

**A:** Технически да, но это может вызвать конфликты маршрутизации. Не рекомендуется.

### Q: Какая производительность при работе через VPN?

**A:** Зависит от:
- Скорости вашего интернета
- Скорости интернета VPN сервера
- Нагрузки на VPN сервер
- Расстояния до VPN сервера

Обычно для REST API задержка увеличивается на 20-50ms, что приемлемо для большинства сценариев.

### Q: Что такое Mirrored Networking и почему это лучше старого способа?

**A:** Mirrored Networking - это новая функция WSL 2.0+, которая делает сеть WSL "зеркальной" с Windows:

**Преимущества:**
- Автоматическая работа с любым VPN без дополнительной настройки
- Нет необходимости вручную настраивать DNS
- WSL и Windows используют одинаковые сетевые интерфейсы
- Проще и надежнее старого метода с метриками и маршрутами

**Старый способ требовал:**
- Ручную настройку `/etc/resolv.conf`
- Изменение метрик сетевых интерфейсов Windows
- Постоянные проблемы при обновлениях

### Q: Безопасно ли использовать публичные DNS (8.8.8.8)?

**A:** С Mirrored Networking вы используете те же DNS что и Windows (включая VPN DNS), поэтому безопасность обеспечивается настройками VPN. Публичные DNS нужны только для старого метода настройки.

### Q: Можно ли автоматизировать запуск проекта при загрузке Windows?

**A:** Да, можно создать задачу в Windows Task Scheduler:

1. Запуск VPN при входе в систему (настраивается в Amnezia клиенте)
2. Запуск WSL и Docker: создайте `.bat` скрипт:

```batch
@echo off
wsl -d Ubuntu -e bash -c "cd /home/main/projects/golang-project && make docker-up"
```

Добавьте его в автозагрузку Windows.

### Q: Что если мне нужно временно отключить статический DNS?

**A:** Просто удалите настройку из `/etc/wsl.conf`:

```bash
sudo nano /etc/wsl.conf
# Удалите секцию [network] или закомментируйте её

# Перезапустите WSL
wsl --shutdown
```

WSL вернется к автогенерации `/etc/resolv.conf`.

---

## Дополнительные ресурсы

- [Документация WSL](https://docs.microsoft.com/en-us/windows/wsl/)
- [Amnezia VPN документация](https://docs.amnezia.org/)
- [Docker в WSL2](https://docs.docker.com/desktop/windows/wsl/)

Если у вас остались вопросы, создайте Issue в репозитории проекта.

