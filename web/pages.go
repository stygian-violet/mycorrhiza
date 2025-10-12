package web

import (
	"embed"

	"github.com/bouncepaw/mycorrhiza/web/newtmpl"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

//go:embed views/*.html
var fs embed.FS

var pageOrphans, pageBacklinks, pageSubhyphae, pageUserList, pageChangePassword *newtmpl.Page
var pageHyphaDelete, pageHyphaRevert, pageHyphaEdit, pageHyphaEmpty, pageHypha *newtmpl.Page
var pageRevision, pageMedia *newtmpl.Page
var pageAuthLogin, pageAuthRegister *newtmpl.Page
var pageCatPage, pageCatList, pageCatEdit *newtmpl.Page

var panelChain, newUserChain, editUserChain, deleteUserChain viewutil.Chain

func initPages() {

	panelChain = viewutil.CopyEnRuWith(fs, "views/admin-panel.html", adminTranslationRu)
	newUserChain = viewutil.CopyEnRuWith(fs, "views/admin-new-user.html", adminTranslationRu)
	editUserChain = viewutil.CopyEnRuWith(fs, "views/admin-edit-user.html", adminTranslationRu)
	deleteUserChain = viewutil.CopyEnRuWith(fs, "views/admin-delete-user.html", adminTranslationRu)

	pageOrphans = newtmpl.NewPage(fs, map[string]string{
		"orphaned hyphae":    "Гифы-сироты",
		"orphan description": "Ниже перечислены гифы без ссылок на них.",
	}, "views/orphans.html")
	pageBacklinks = newtmpl.NewPage(fs, map[string]string{
		"backlinks to text": `Обратные ссылки на {{.}}`,
		"backlinks to link": `Обратные ссылки на <a href="{{.Meta.Root}}hypha/{{.HyphaName}}">{{beautifulName .HyphaName}}</a>`,
		"description":       `Ниже перечислены гифы, на которых есть ссылка на эту гифу, трансклюзия этой гифы или эта гифа вставлена как изображение.`,
	}, "views/backlinks.html")
	pageSubhyphae = newtmpl.NewPage(fs, map[string]string{
		"subhyphae of": `Подгифы`,
	}, "views/subhyphae.html")
	pageUserList = newtmpl.NewPage(fs, map[string]string{
		"user list":     "Список пользователей",
		"manage users":  "Управление пользователями",
		"create user":   "Создать пользователя",
		"reindex users": "Переиндексировать пользователей",
		"name":          "Имя",
		"group":         "Группа",
		"registered at": "Зарегистрирован",
		"actions":       "Действия",
		"edit":          "Изменить",
	}, "views/user-list.html")
	pageChangePassword = newtmpl.NewPage(fs, map[string]string{
		"change password":           "Сменить пароль",
		"confirm password":          "Повторите пароль",
		"current password":          "Текущий пароль",
		"non local password change": "Пароль можно поменять только местным аккаунтам. Telegram-аккаунтам нельзя.",
		"password":                  "Пароль",
		"submit":                    "Поменять",
	}, "views/change-password.html")
	pageHyphaDelete = newtmpl.NewPage(fs, map[string]string{
		"delete hypha?":      "Удалить {{beautifulName .}}?",
		"delete [[hypha]]?":  "Удалить <a href=\"{{.Meta.Root}}hypha/{{.HyphaName}}\">{{beautifulName .HyphaName}}</a>?",
		"want to delete?":    "Вы действительно хотите удалить эту гифу?",
		"delete recursively": "Также удалить подгифы",
	}, "views/hypha-delete.html")
	pageHyphaRevert = newtmpl.NewPage(fs, map[string]string{
		"revert":            "Откатить",
		"to revision":       "к ревизии",
		"want to revert?":   "Вы действительно хотите откатить эту гифу?",
	}, "views/hypha-revert.html")
	pageHyphaEdit = newtmpl.NewPage(fs, map[string]string{
		"editing hypha":               `Редактирование {{beautifulName .}}`,
		"editing [[hypha]]":           `Редактирование <a href="{{.Meta.Root}}hypha/{{.HyphaName}}">{{beautifulName .HyphaName}}</a>`,
		"creating [[hypha]]":          `Создание <a href="{{.Meta.Root}}hypha/{{.HyphaName}}">{{beautifulName .HyphaName}}</a>`,
		"you're creating a new hypha": `Вы создаёте новую гифу.`,
		"describe your changes":       `Опишите ваши правки`,
		"save":                        `Сохранить`,
		"preview":                     `Предпросмотр`,
		"previewing hypha":            `Предпросмотр {{beautifulName .}}`,
		"preview tip":                 `Заметьте, эта гифа ещё не сохранена. Вот её предпросмотр:`,

		"markup":             `Разметка`,
		"link":               `Ссылка`,
		"link title":         `Текст`,
		"heading":            `Заголовок`,
		"bold":               `Жирный`,
		"italic":             `Курсив`,
		"highlight":          `Выделение`,
		"underline":          `Подчеркивание`,
		"mono":               `Моноширинный`,
		"super":              `Надстрочный`,
		"sub":                `Подстрочный`,
		"strike":             `Зачёркнутый`,
		"rocket":             `Ссылка-ракета`,
		"transclude":         `Трансклюзия`,
		"hr":                 `Гориз. черта`,
		"code":               `Код-блок`,
		"bullets":            `Маркир. список`,
		"numbers":            `Нумер. список`,
		"mycomarkup help":    `<a href="{{.Meta.Root}}help/en/mycomarkup" class="shy-link">Подробнее</a> о Микоразметке`,
		"actions":            `Действия`,
		"current date local": `Местная дата`,
		"current time local": `Местное время`,
		"current date utc":   "Дата UTC",
		"current time utc":   "Время UTC",
		"selflink":           `Ссылка на вас`,
	}, "views/hypha-edit.html")
	pageHypha = newtmpl.NewPage(fs, map[string]string{
		"edit text":     "Редактировать",
		"log out":       "Выйти",
		"admin panel":   "Админка",
		"subhyphae":     "Подгифы",
		"history":       "История",
		"rename":        "Переименовать",
		"delete":        "Удалить",
		"view markup":   "Посмотреть разметку",
		"manage media":  "Медиа",
		"turn to media": "Превратить в медиа-гифу",
		"backlinks":     "{{.BacklinkCount}} обратн{{if eq .BacklinkCount 1}}ая ссылка{{else if and (le .BacklinkCount 4) (gt .BacklinkCount 1)}}ые ссылки{{else}}ых ссылок{{end}}",
		"subhyphae link":"подгифы",

		"empty heading":                    `Эта гифа не существует`,
		"empty no rights":                  `У вас нет прав для создания новых гиф. Вы можете:`,
		"empty log in":                     `Войти в свою учётную запись, если она у вас есть`,
		"empty register":                   `Создать новую учётную запись`,
		"write a text":                     `Написать текст`,
		"write a text tip":                 `Напишите заметку, дневник, статью, рассказ или иной текст с помощью <a href="{{.Meta.Root}}help/en/mycomarkup" class="shy-link">Микоразметки</a>. Сохраняется полная история правок документа.`,
		"write a text writing conventions": `Не забывайте следовать правилам оформления этой вики, если они имеются.`,
		"write a text btn":                 `Создать`,
		"upload a media":                   `Загрузить медиа`,
		"upload a media tip":               `Загрузите изображение, видео или аудио. Распространённые форматы можно просматривать из браузера, остальные можно только скачать и просмотреть локально. Позже вы можете дописать пояснение к этому медиа.`,
		"upload a media btn":               `Загрузить`,
	}, "views/hypha.html")
	pageRevision = newtmpl.NewPage(fs, map[string]string{
		"revert":           "Откатить",
		"revision link":    "Посмотреть Микоразметку",
		"hypha at rev":     "{{.HyphaName}} на {{.RevHash}}",
	}, "views/hypha-revision.html")
	pageMedia = newtmpl.NewPage(fs, map[string]string{ // TODO: сделать новый перевод
		"media title":    "Медиа «{{.HyphaName | beautifulLink}}»",
		"tip":            "На этой странице вы можете управлять медиа.",
		"empty":          "Эта гифа не имеет медиа, здесь вы можете его загрузить.",
		"what is media?": "Что такое медиа?",
		"stat":           "Свойства",
		"stat size":      "Размер файла:",
		"stat mime":      "MIME-тип:",

		"upload title": "Прикрепить",
		"upload tip":   "Вы можете загрузить новое медиа. Пожалуйста, не загружайте слишком большие изображения без необходимости, чтобы впоследствии не ждать её долгую загрузку.",
		"upload btn":   "Загрузить",

		"remove title": "Открепить",
		"remove tip":   "Заметьте, чтобы заменить медиа, вам не нужно его перед этим откреплять.",
		"remove btn":   "Открепить",
	}, "views/hypha-media.html")

	pageAuthLogin = newtmpl.NewPage(fs, map[string]string{
		"username":       "Логин",
		"password":       "Пароль",
		"log in":         "Войти",
		"log out":        "Выйти",
		"approval tip":   "Новые пользователи должны быть одобрены администратором, прежде чем они смогут получить доступ к вики.",
		"cookie tip":     "Отправляя эту форму, вы разрешаете вики хранить cookie в вашем браузере. Это позволит движку связывать ваши правки с вашей учётной записью. Вы будете авторизованы, пока не выйдете из учётной записи.",
		"log in to x":    "Войти в {{.}}",
		"lock title":     "🔒 Доступ закрыт",
		"error":          "Ошибка",
		"error login":    "Неправильное имя пользователя или пароль.",
		"error telegram": "Не удалось войти через Телеграм.",
		"register":       "Регистрация",
	}, "views/auth-base.html", "views/auth-telegram.html", "views/auth-login.html")

	pageAuthRegister = newtmpl.NewPage(fs, map[string]string{
		"username":      "Логин",
		"password":      "Пароль",
		"approval tip":  "Новые пользователи должны быть одобрены администратором, прежде чем они смогут получить доступ к вики.",
		"cookie tip":    "Отправляя эту форму, вы разрешаете вики хранить cookie в вашем браузере. Это позволит движку связывать ваши правки с вашей учётной записью. Вы будете авторизованы, пока не выйдете из учётной записи.",
		"password tip":  "Сервер хранит ваш пароль в зашифрованном виде, даже администраторы не смогут его прочесть.",
		"error":         "Ошибка",
		"register btn":  "Зарегистрироваться",
		"register on x": "Регистрация на {{.}}",
	}, "views/auth-base.html", "views/auth-telegram.html", "views/auth-register.html")

	pageCatPage = newtmpl.NewPage(fs, map[string]string{
		"category x": "Категория {{. | beautifulName}}",
		"edit":       "Редактировать",
		"cat":        "Категория",
		"empty cat":  "Эта категория пуста.",
	}, "views/cat-page.html")

	pageCatEdit = newtmpl.NewPage(fs, map[string]string{
		"edit category x":       "Редактирование категории {{beautifulName .}}",
		"edit category heading": "Редактирование категории <a href=\"{{.Meta.Root}}category/{{.CatName}}\">{{beautifulName .CatName}}</a>",
		"empty cat":             "Эта категория пуста.",
		"add to category title": "Добавить гифу в эту категорию",
		"hypha name":            "Название гифы",
		"add":                   "Добавить",
		"remove hyphae":         "Убрать гифы из этой категории",
		"remove":                "Убрать",
	}, "views/cat-edit.html")

	pageCatList = newtmpl.NewPage(fs, map[string]string{
		"category list": "Список категорий",
		"no categories": "В этой вики нет категорий.",
	}, "views/cat-list.html")
}
