# sr-rules-gen

инструмент для перевода geosite/geoip в списки правил формата ShadowRocket

в этой репе настроен воркфлоу, который регулярно подтягивает и парсит файлы с https://github.com/runetfreedom/russia-v2ray-rules-dat и генерирует списки в ветке [release](https://github.com/cerdocyonina/sr-rules-gen/tree/release):

- [geosite](https://github.com/cerdocyonina/sr-rules-gen/tree/release/geosite)
- [geoip](https://github.com/cerdocyonina/sr-rules-gen/tree/release/geoip)

## использование в ShadowRocket

в конфиге ShadowRocket:

```
# geosite
RULE-SET,https://raw.githubusercontent.com/cerdocyonina/sr-rules-gen/release/geosite/refilter.list,PROXY

# geoip
RULE-SET,https://raw.githubusercontent.com/cerdocyonina/sr-rules-gen/release/geoip/ru-blocked.list,PROXY
```


## gen-rules

общая утилита

флаги:

```
-geoip-dir string
      geoip output directory (default "dist/geoip")
-geoip-url string
      geoip file path/url (default "https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat")
-geosite-dir string
      geosite output directory (default "dist/geosite")
-geosite-url string
      geosite file path/url (default "https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat")
-workers int
      workers count to use (default 8)
```

примеры:
  
1. парсинг из url:
    ```bash
    ./gen-rules -geosite-url https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat
    ```

1. парсинг из локальных файлов:
    ```bash
    ./gen-rules -geosite-url ./geosite.dat
    ```

1. генерация списков в директории:
    ```bash
    ./gen-rules -geosite-dir ./geosite -geoip-dir ./geoip
    ```

    > создаст полный путь к директории, если не существует (типа mkdir -p)

