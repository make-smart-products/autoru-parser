# autoru-parser

Парсер объявлений [auto.ru](https://auto.ru) на Go: извлекает характеристики автомобиля и скачивает фотографии.

## Возможности

- Парсинг страницы объявления (владение, базовые характеристики, цена, продавец)
- Загрузка полных технических характеристик через внутренний API каталога
- Извлечение всех фото из HTML (URL в максимальном разрешении 1200×900)
- Сохранение результата в JSON и скачивание фотографий на диск
- Плоский словарь всех характеристик через `listing.AllSpecs()`

## Установка

```bash
git clone https://github.com/make-smart-products/autoru-parser.git
cd autoru-parser
go build -o autoru-parser .
```

## Использование

```bash
autoru-parser -url "https://auto.ru/cars/used/sale/mazda/3/1133169996-dbffd21a/"
```

### Флаги

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `-url` | — | URL страницы объявления (обязательный) |
| `-out` | `output` | Папка для JSON и фото |
| `-no-photos` | `false` | Не скачивать фотографии |
| `-timeout` | `2m` | Таймаут запросов |
| `-pretty` | `true` | Форматированный JSON |

### Результат

```
output/
├── listing.json    # все данные
└── photos/
    ├── 001.jpg
    └── ...
```

## Пример как библиотеки

```go
client := parser.NewClient(parser.WithTimeout(2 * time.Minute))
listing, err := client.Parse(ctx, url)
paths, err := client.DownloadPhotos(ctx, listing, "photos")
specs := listing.AllSpecs()
```

## Тесты

```bash
go test ./...
```

## Ограничения

- auto.ru может показать капчу при частых запросах
- Список опций комплектации (ABS, климат и т.д.) частично доступен только через JS на сайте
- Технические характеристики из каталога загружаются, когда в HTML есть ссылка на спецификацию

## Лицензия

MIT
