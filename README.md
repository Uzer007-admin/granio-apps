# granio-apps

Компонент независимого проекта **Granio — Гранио**.

**Исходный репозиторий:** [apps](https://github.com/fyne-io/apps).
**Цель форка:** Каталог приложений Granio и проверка совместимости ядра.

История изменений, лицензии и сведения об исходных авторах сохранены.
Go-модули переведены на собственные пути `github.com/Uzer007-admin/…`;
минимальная версия Go — **1.27.1**. Ядро и общие API подключаются из
[`granio/v2`](https://github.com/Uzer007-admin/granio).

Состояние переноса и результаты проверок приведены в
[реестре экосистемы](https://github.com/Uzer007-admin/granio/blob/dev/granio/docs/ECOSYSTEM.md).
Наличие форка само по себе не означает проверку всех платформ и сценариев.

---

## Документация исходного проекта

Команды установки и ссылки в сохранённом тексте ниже относятся к исходнику.
Для Granio используются новые пути модулей, перечисленные выше.

# Fyne Apps Installer

This app is designed to help you browse and install available fyne apps.
It loads data using the [apps.fyne.io](https://apps.fyne.io) API to stay up to date.

This app builds all others from source code, so it can take some time.
It will also require some setup, shown in *Installing*, below.

## Installing

The easiest way to install this app is using the `fyne` command, and it's `install` utility.
To do so you will need to have Go and C compilers installed - if you need help you can see the [fyne prerequisites](https://developer.fyne.io/started/).
Once set up execute the following

```
$ go install fyne.io/tools/v2/cmd/fyne@latest
$ fyne install fyne.io/apps
```

The commands above will install the `Apps` app in your standard system location.
You can then launch it like any other app.

## Running

Once installed launch the app from your app launcher, file browser or programme manager by tapping on the icon:

<p align="center">
<img src="/Icon.png" width="128" />
</p>

When loaded you should see something like the following screenshot:

![](/img/screenshot.png)

Browse the apps, find one you like and tap the *Install* button.
Installed apps will appear alongside this app in the standard system location.
