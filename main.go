package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	domainsDir  = "domains"
	validDir    = "valid"
	invalidDir  = "invalid"
	dialTimeout = 5 * time.Second
)

func main() {
	// Парсинг количества потоков из аргументов командной строки
	threads := flag.Int("threads", 1, "number of threads to use")
	flag.Parse()

	// Создание папок для валидных и невалидных доменов, если они не существуют
	createDirIfNotExist(validDir)
	createDirIfNotExist(invalidDir)

	// Создание канала для доменов
	domains := make(chan string)

	// Создание WaitGroup для ожидания завершения всех горутин
	var wg sync.WaitGroup

	// Запуск горутин
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for domain := range domains {
				checkDomain(domain)
			}
		}()
	}

	// Получение списка файлов в папке domains
	files, err := ioutil.ReadDir(domainsDir)
	if err != nil {
		fmt.Println("Error reading domains directory:", err)
		return
	}

	// Выбор случайного файла и случайной строки из этого файла
	rand.Seed(time.Now().UnixNano())
	randomFileIndex := rand.Intn(len(files))
	fileName := files[randomFileIndex].Name()

	// Чтение и удаление случайной строки из файла
	domain, err := readAndRemoveRandomLine(filepath.Join(domainsDir, fileName))
	if err != nil {
		fmt.Println("Error reading/removing line from file:", err)
		return
	}

	// Отправка домена в канал
	domains <- domain

	// Закрытие канала и ожидание завершения всех горутин
	close(domains)
	wg.Wait()
}

// checkDomain проверяет домен на валидность и записывает результат в соответствующий файл
func checkDomain(domain string) {
	conn, err := net.DialTimeout("tcp", domain+":80", dialTimeout)
	if err != nil {
		writeToFile(filepath.Join(invalidDir, "invalid.txt"), domain+"\n")
	} else {
		conn.Close()
		writeToFile(filepath.Join(validDir, "valid.txt"), domain+"\n")
	}
}

// readAndRemoveRandomLine читает и удаляет случайную строку из файла
func readAndRemoveRandomLine(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("file is empty")
	}

	randomLineIndex := rand.Intn(len(lines))
	selectedLine := lines[randomLineIndex]

	// Удаление выбранной строки
	lines = append(lines[:randomLineIndex], lines[randomLineIndex+1:]...)

	// Перезапись файла без выбранной строки
	err = writeLinesToFile(filePath, lines)
	if err != nil {
		return "", err
	}

	return selectedLine, nil
}

// writeToFile записывает строку в файл
func writeToFile(filePath, text string) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(text); err != nil {
		fmt.Println("Error writing to file:", err)
	}
}

// writeLinesToFile записывает строки в файл
func writeLinesToFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// createDirIfNotExist создает папку, если она не существует
func createDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
		}
	}
}
