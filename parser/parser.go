package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"vak-parser/common"
	"vak-parser/database"
)

var (
	logAPI  = log.New(os.Stderr, "[api] ", log.LstdFlags)
	logDL   = log.New(os.Stderr, "[download] ", log.LstdFlags)
	logPDF  = log.New(os.Stderr, "[pdf] ", log.LstdFlags)
	logOCR  = log.New(os.Stderr, "[ocr] ", log.LstdFlags)
	logMain = log.New(os.Stderr, "[main] ", log.LstdFlags)
)

func Parse(botIn chan<- common.BotMsg, botOut <-chan common.BotMsg, schedCh <-chan struct{},db *database.DbAdapter) {
	dlCh := make(chan Result, 10)
	pdfCh := make(chan ChannelPayload, 20)
	imgCh := make(chan ChannelPayload, 30)
	sucCh := make(chan common.SuccessPayload, 5)

	for range 10 {
		go downloadFile(dlCh, pdfCh)
	}
	for range 20 {
		go parsePdf(pdfCh, imgCh)
	}

	rawKeyWord, keyWordPattern := getKeyWordPattern()
	logMain.Printf("Поиск ключевого слова %s", rawKeyWord)

	for range 30 {
		go parseImg(imgCh, sucCh, keyWordPattern)
	}

	for {
		select {
			case payload :=<-sucCh:
				botIn <-common.BotMsg {
					Type: common.BotMsgTypeSuccess,
					SuccessPayload: payload,
				}
			case msg :=<-botOut:
				if msg.Type == common.BotMsgTypeParse {
					data, err := parsePage()
					if err != nil {
						logMain.Printf("ошибка загрузки страницы: %v", err)
						continue
					}

					parseResults(data, db, dlCh, sucCh)
				}
			case <-schedCh:
				data, err := parsePage()
				if err != nil {
					logMain.Printf("ошибка загрузки страницы: %v", err)
					continue
				}

				parseResults(data, db, dlCh, sucCh)
		}
	}

}

func parseImg(imgCh chan ChannelPayload, sucCh chan common.SuccessPayload, keyPattern *regexp.Regexp) {
	for data := range imgCh {
		logOCR.Printf("распознавание: %s", data.FilePath)

		cmd := exec.Command("tesseract", data.FilePath, "stdout", "-l", "rus+eng")
		out, err := cmd.Output()
		if err != nil {
			logOCR.Printf("ошибка OCR для %s: %v", data.FilePath, err)
			continue
		}

		text := strings.ToLower(string(out))
		if keyPattern.MatchString(text) {
			re := regexp.MustCompile(`[\w\/]+-page-(\d+)\.png`)
			pageStr := re.FindStringSubmatch(data.FilePath)[1]
			pageNum, err := strconv.Atoi(pageStr)
			if err != nil {
				pageNum = -1
			}

			sucCh <- common.SuccessPayload{
				Url: data.Url,
				Page: pageNum,
			}
			logOCR.Printf("ключевое слово найдено на странице %s %s", pageStr, data.Url)
		}

		if err := os.Remove(data.FilePath); err != nil {
			logOCR.Printf("не удалось удалить %s: %v", data, err)
		}
	}
	logOCR.Println("воркер завершил работу")
}

func parsePdf(pdfCh chan ChannelPayload, imgCh chan ChannelPayload) {
	for data := range pdfCh {
		logPDF.Printf("конвертация PDF: %s", data.FilePath)

		pdfNum, _, found := strings.Cut(data.FilePath, ".")
		if !found {
			logPDF.Printf("не удалось определить имя файла: %s", data.FilePath)
			continue
		}

		imgNamePattern := fmt.Sprintf("%s-page", pdfNum)
		cmd := exec.Command("pdftoppm", "-png", "-r", "150", data.FilePath, imgNamePattern)
		if err := cmd.Run(); err != nil {
			logPDF.Printf("ошибка pdftoppm для %s: %v", data.FilePath, err)
			continue
		}

		files, err := filepath.Glob(fmt.Sprintf("%s-page-*.png", pdfNum))
		if err != nil {
			logPDF.Printf("ошибка поиска PNG для %s: %v", data.FilePath, err)
			continue
		}

		logPDF.Printf("PDF %s: найдено %d страниц", data.FilePath, len(files))
		for _, imgPath := range files {
			imgCh <- ChannelPayload{
				FilePath: imgPath,
				Url: data.Url,
			}
		}

		if err := os.Remove(data.FilePath); err != nil {
			logPDF.Printf("не удалось удалить %s: %v", data.FilePath, err)
		}
	}
	logPDF.Println("воркер завершил работу")
}

func downloadFile(dlCh chan Result, pdfCh chan ChannelPayload) {
	for data := range dlCh {

		pdfUrl := &data.Files[0].Url
		pdfGuid := &data.Files[0].Name
		logDL.Printf("[%s] скачивание: %s", *pdfGuid, *pdfUrl)

		resp, err := http.Get(*pdfUrl)
		if err != nil {
			logDL.Printf("[%s] ошибка запроса: %v", *pdfGuid, err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logDL.Printf("[%s] ошибка чтения ответа: %v", *pdfGuid, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logDL.Printf("[%s] HTTP %d для %s", *pdfGuid, resp.StatusCode, *pdfUrl)
			continue
		}

		tmpFileName := fmt.Sprintf("%d.pdf", pdfGuid)
		tmpFile, err := os.CreateTemp("", tmpFileName)
		if err != nil {
			logDL.Printf("[%s] ошибка создания временного файла: %v", *pdfGuid, err)
			continue
		}

		if _, err := tmpFile.Write(body); err != nil {
			logDL.Printf("[%s] ошибка записи %s: %v", *pdfGuid, tmpFile.Name(), err)
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			continue
		}

		if err := tmpFile.Close(); err != nil {
			logDL.Printf("[%s] ошибка закрытия %s: %v", *pdfGuid, tmpFile.Name(), err)
		}

		logDL.Printf("[%s] сохранён %s (%d байт), отправка на конвертацию", *pdfGuid, tmpFile.Name(), len(body))
		pdfCh <- ChannelPayload { Url: *pdfUrl ,FilePath: tmpFile.Name() }
	}
	logDL.Println("воркер завершил работу")
}

func getKeyWordPattern() (string, *regexp.Regexp) {
	keyWord := strings.ToLower(os.Getenv("KEYWORD"))
	if keyWord == "" {
		keyWord = "баранов"
	}
	return keyWord, regexp.MustCompile(fmt.Sprintf(`%s[^а-я]`, keyWord))
}

func parseResults(data Data, db *database.DbAdapter, dlCh chan Result, sucCh chan common.SuccessPayload) {
	logMain.Printf("запуск пайплайна: %d результатов", len(data.Results))

	rawKeyWord, _ := getKeyWordPattern()

	queued := 0
	for idx, result := range data.Results {
		inCache, err := db.HasResult(result.Id)
		if err != nil {
			logMain.Printf("ошибка кэша для guid=%s: %v", result.Id, err)
			continue
		}
		if inCache {
			logMain.Printf("Айтем с guid=%s есть в кэше. Пропуск...", result.Id)
			continue
		}

		for _, keyword := range result.Keywords {
			if strings.Contains(keyword.Name, rawKeyWord) {
				logOCR.Printf("ключевое слово найдено в тегах по урлу: %s", result.Files[0].Url)
				sucCh<-common.SuccessPayload{
					Url: result.Files[0].Url,
					Page: -1,
				}
			}
		}

		if len(result.Files) == 0 {
			logMain.Printf("[%d] пропуск: нет файлов (%s)", idx, result.Name)
			continue
		}

		logMain.Printf("[%d] в очередь: %s — %s", idx, result.Name, result.Files[0].Url)
		dlCh <- result
		queued++

		if err := db.MarkResult(result.Id); err != nil {
			logMain.Printf("ошибка записи в кэш guid=%s: %v", result.Id, err)
		}
	}

	logMain.Println("пайплайн завершён")
}

func parsePage() (Data, error) {
	const url = "https://vak.gisnauka.ru/api/news/news-list/?page=1&pageSize=10&type=8,14"
	logAPI.Printf("запрос: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return Data{}, fmt.Errorf("запрос списка новостей: %w", err)
	}
	defer resp.Body.Close()

	logAPI.Printf("ответ: HTTP %d", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Data{}, fmt.Errorf("чтение ответа: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Data{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var jsonData Data
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return Data{}, fmt.Errorf("разбор JSON: %w", err)
	}

	logAPI.Printf("получено записей: %d", len(jsonData.Results))
	return jsonData, nil
}
