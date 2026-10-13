package communications

import (
	"fmt"
	"time"
)

func Template(kind, locale, link string, count int) (string, string) {
	return BrandedTemplate(kind, locale, link, count, "FileX")
}
func BrandedTemplate(kind, locale, link string, count int, brand string) (string, string) {
	if locale == "en" {
		switch kind {
		case "transfer.published":
			return "Files are ready", "You have received files. Open this private link and select Download:\n\n" + link + "\n\nEach file can be downloaded once. Do not share the link."
		case "transfer.expiring":
			return "File access expires soon", "Your transfer will expire soon. Review it in " + brand + "."
		case "transfer.expired":
			return "Transfer expired", "Access to the transfer has expired."
		case "download.completed":
			return "Download completed", "Files were downloaded. The journal contains aggregate data without recipient identification."
		case "digest":
			return brand + " activity", fmt.Sprintf("There are %d new events. View details in %s.", count, brand)
		}
		return brand + " notification", "Open " + brand + " to review your transfer."
	}
	switch kind {
	case "transfer.published":
		return "Файлы готовы к получению", "Вам передали файлы. Откройте личную ссылку и нажмите «Скачать»:\n\n" + link + "\n\nКаждый файл можно получить один раз. Не передавайте ссылку другим."
	case "transfer.expiring":
		return "Срок доступа скоро закончится", "Срок передачи скоро закончится. Проверьте её в " + brand + "."
	case "transfer.expired":
		return "Срок передачи закончился", "Доступ к передаче закрыт по истечении срока."
	case "download.completed":
		return "Скачивание завершено", "Файлы скачаны. В журнале доступны общие данные без идентификации получателя."
	case "digest":
		return "События " + brand, fmt.Sprintf("Новых событий: %d. Подробности доступны в %s.", count, brand)
	}
	return "Уведомление " + brand, "Откройте " + brand + ", чтобы проверить передачу."
}
func Quiet(at time.Time, timezone string, start, end *int) bool {
	if start == nil || end == nil || *start == *end {
		return false
	}
	loc, e := time.LoadLocation(timezone)
	if e != nil {
		return true
	}
	h := at.In(loc).Hour()
	if *start < *end {
		return h >= *start && h < *end
	}
	return h >= *start || h < *end
}
