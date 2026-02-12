// Пакет main - точка входа для симулятора устройств
// Симулирует работу 20,000 устройств, отправляющих TR181 данные через Apache Pulsar
package main

import (
	"context"      // Контекст для отправки сообщений в Pulsar
	"encoding/json" // Сериализация данных в JSON
	"fmt"          // Форматирование строк (серийные номера)
	"log"          // Логирование
	"math/rand"    // Генерация случайных чисел
	"os"           // Переменные окружения
	"os/signal"    // Обработка сигналов завершения
	"sync"         // sync.WaitGroup для ожидания горутин
	"syscall"      // SIGINT, SIGTERM
	"time"         // Интервалы, таймеры, время

	pulsarclient "github.com/apache/pulsar-client-go/pulsar" // Pulsar клиент
	"golang-test-dev/pkg/pulsar"                              // Константы тем
	"golang-test-dev/pkg/tr181"                               // Модель TR181
)

// Константы симулятора
const (
	numDevices = 20000              // Количество симулируемых устройств
	interval   = 30 * time.Second   // Интервал отправки данных от каждого устройства (30 сек)
	batchSize  = 50                 // Размер батча перед отправкой в Pulsar
)

// Simulator - основной объект симулятора
type Simulator struct {
	devices   []Device                // Список всех устройств
	client    pulsarclient.Client     // Pulsar клиент
	producer  pulsarclient.Producer   // Producer для публикации в Pulsar
	wg        sync.WaitGroup          // Счётчик горутин (для корректной остановки)
	stopChan  chan struct{}           // Канал сигнала остановки (закрывается при Stop)
	batchChan chan tr181.TR181Device  // Канал для сбора данных в батч
}

// Device - параметры одного симулируемого устройства
type Device struct {
	SerialNumber string // Серийный номер (DEV-00000001, DEV-00000002, ...)
	baseCPU      int    // Базовое значение CPU для генерации вариаций
	baseMemory   int    // Базовое значение памяти
	baseWiFi2GHz int    // Базовый сигнал WiFi 2.4 GHz (dBm)
	baseWiFi5GHz int    // Базовый сигнал WiFi 5 GHz (dBm)
	baseWiFi6GHz int    // Базовый сигнал WiFi 6 GHz (dBm)
}

// NewSimulator - создаёт симулятор и подключается к Pulsar
func NewSimulator(pulsarURL string) (*Simulator, error) {
	// Подключаемся к Pulsar
	client, err := pulsar.NewClient(pulsarURL)
	if err != nil {
		return nil, fmt.Errorf("pulsar client: %w", err)
	}

	// Создаём producer для публикации в тему tr181-device-data
	producer, err := client.CreateProducer(pulsarclient.ProducerOptions{
		Topic: pulsar.TopicTR181Data,
		Name:  "simulator-producer",
	})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("pulsar producer: %w", err)
	}

	// Инициализируем структуру симулятора
	return &Simulator{
		devices:   make([]Device, numDevices),           // Массив под 20000 устройств
		client:    client,
		producer:  producer,
		stopChan:  make(chan struct{}),                 // Канал без буфера
		batchChan: make(chan tr181.TR181Device, batchSize*10), // Буфер на 500 сообщений
	}, nil
}

// Close - освобождает ресурсы Pulsar
func (s *Simulator) Close() error {
	s.producer.Close()
	s.client.Close()
	return nil
}

// Init - инициализирует все устройства со случайными базовыми параметрами
func (s *Simulator) Init() {
	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())
	// Для каждого устройства задаём уникальные параметры
	for i := 0; i < numDevices; i++ {
		s.devices[i] = Device{
			SerialNumber: fmt.Sprintf("DEV-%08d", i+1), // DEV-00000001 ... DEV-00020000
			baseCPU:      30 + rand.Intn(30),           // CPU 30-60%
			baseMemory:   40 + rand.Intn(30),           // Память 40-70%
			baseWiFi2GHz: -70 + rand.Intn(20),          // WiFi 2.4 GHz: -70..-50 dBm
			baseWiFi5GHz: -75 + rand.Intn(20),          // WiFi 5 GHz: -75..-55 dBm
			baseWiFi6GHz: -80 + rand.Intn(20),          // WiFi 6 GHz: -80..-60 dBm
		}
	}
	log.Printf("Initialized %d devices", numDevices)
}

// Start - запускает симуляцию
func (s *Simulator) Start() {
	log.Println("Starting simulator...")
	// Добавляем 1 в WaitGroup для batchProcessor
	s.wg.Add(1)
	// Запускаем обработчик батчей в отдельной горутине
	go s.batchProcessor()

	// Для каждого устройства запускаем горутину симуляции
	for i := range s.devices {
		s.wg.Add(1)
		go s.simulateDevice(&s.devices[i])
	}
	log.Println("Simulator started (publishing to Apache Pulsar)")
}

// Stop - останавливает все горутины и ждёт их завершения
func (s *Simulator) Stop() {
	close(s.stopChan) // Закрываем канал - все горутины получают сигнал
	s.wg.Wait()       // Ждём завершения всех горутин
	log.Println("Simulator stopped")
}

// simulateDevice - симулирует одно устройство (работает в отдельной горутине)
func (s *Simulator) simulateDevice(device *Device) {
	defer s.wg.Done() // По завершении уменьшаем счётчик WaitGroup
	// Случайная задержка 0-1000 мс - распределяем нагрузку во времени
	initialDelay := time.Duration(rand.Intn(1000)) * time.Millisecond
	time.Sleep(initialDelay)

	// Таймер срабатывает каждые 30 секунд
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Бесконечный цикл до получения сигнала остановки
	for {
		select {
		case <-s.stopChan:
			// Получен сигнал остановки - выходим
			return
		case <-ticker.C:
			// Прошло 30 секунд - генерируем и отправляем данные
			s.generateAndQueueData(device)
			// Небольшая задержка 0-10 мс для распределения
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		}
	}
}

// generateAndQueueData - генерирует данные устройства и ставит в очередь на батч
func (s *Simulator) generateAndQueueData(device *Device) {
	// Генерируем данные с вариациями от базовых значений
	data := tr181.DeviceData{
		CPUUsage:                 s.vary(device.baseCPU, 10),      // CPU ±10%
		MemoryUsage:              s.vary(device.baseMemory, 10),   // Память ±10%
		CPUTemperature:           45 + rand.Intn(15),              // 45-60°C
		BoardTemperature:         40 + rand.Intn(10),              // 40-50°C
		RadioTemperature:         50 + rand.Intn(15),              // 50-65°C
		WiFi2GHzSignalStrength:   s.vary(device.baseWiFi2GHz, 10),
		WiFi5GHzSignalStrength:   s.vary(device.baseWiFi5GHz, 10),
		WiFi6GHzSignalStrength:   s.vary(device.baseWiFi6GHz, 10),
		EthernetBytesSent:        int64(rand.Intn(1000000)),
		EthernetBytesReceived:    int64(rand.Intn(1000000)),
		Uptime:                   int64(rand.Intn(86400 * 30)),   // До 30 дней в секундах
	}

	// 5% вероятность - высокий CPU (для генерации алерта high-cpu-usage)
	if rand.Float32() < 0.05 {
		data.CPUUsage = 65 + rand.Intn(30) // 65-95%
	}
	// 3% вероятность - слабый WiFi (для генерации алерта low-wifi)
	if rand.Float32() < 0.03 {
		data.WiFi2GHzSignalStrength = -105 + rand.Intn(5) // -105..-100 dBm
	}

	// Формируем полную структуру TR181Device
	deviceData := tr181.TR181Device{
		SerialNumber: device.SerialNumber,
		Timestamp:    time.Now(),
		Data:         data,
	}

	// Отправляем в канал (неблокирующая отправка через select)
	select {
	case s.batchChan <- deviceData:
		// Успешно добавлено в очередь
	case <-s.stopChan:
		// Сигнал остановки - выходим
		return
	}
}

// batchProcessor - собирает данные в батчи и отправляет в Pulsar
func (s *Simulator) batchProcessor() {
	defer s.wg.Done()
	// Таймер - отправляем неполный батч каждую секунду
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Накопленный батч
	var batch []tr181.TR181Device

	for {
		select {
		case <-s.stopChan:
			// Остановка - отправляем оставшиеся данные
			if len(batch) > 0 {
				s.publishBatch(batch)
			}
			return
		case deviceData := <-s.batchChan:
			// Получили данные от устройства
			batch = append(batch, deviceData)
			// Если батч достиг размера - отправляем
			if len(batch) >= batchSize {
				s.publishBatch(batch)
				batch = batch[:0] // Очищаем слайс
			}
		case <-ticker.C:
			// Прошла секунда - отправляем накопленный батч (даже неполный)
			if len(batch) > 0 {
				s.publishBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// publishBatch - отправляет батч данных в Pulsar
func (s *Simulator) publishBatch(batch []tr181.TR181Device) {
	if len(batch) == 0 {
		return
	}

	log.Printf("Publishing batch of %d devices to Pulsar", len(batch))
	successCount := 0
	// Отправляем каждое устройство отдельным сообщением
	for _, deviceData := range batch {
		// Сериализуем в JSON
		jsonData, err := json.Marshal(deviceData)
		if err != nil {
			log.Printf("Failed to marshal data for %s: %v", deviceData.SerialNumber, err)
			continue
		}

		// Публикуем сообщение в Pulsar
		_, err = s.producer.Send(context.Background(), &pulsarclient.ProducerMessage{
			Payload: jsonData,
			Key:     deviceData.SerialNumber, // Ключ для партиционирования по устройству
		})
		if err != nil {
			log.Printf("Failed to publish %s: %v", deviceData.SerialNumber, err)
			continue
		}
		successCount++
	}

	if successCount > 0 {
		log.Printf("Batch published: %d/%d devices", successCount, len(batch))
	}
}

// vary - добавляет случайную вариацию ±variance к базовому значению
func (s *Simulator) vary(base, variance int) int {
	val := base + rand.Intn(variance*2) - variance
	if val < 0 {
		val = 0
	}
	return val
}

func main() {
	// URL Pulsar из переменной окружения
	pulsarURL := os.Getenv("PULSAR_URL")
	if pulsarURL == "" {
		pulsarURL = "pulsar://localhost:6650"
	}

	// Создаём симулятор
	simulator, err := NewSimulator(pulsarURL)
	if err != nil {
		log.Fatalf("Failed to create simulator: %v", err)
	}
	defer simulator.Close()

	// Инициализируем устройства
	simulator.Init()
	// Запускаем симуляцию
	simulator.Start()

	// Ожидание сигнала завершения (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down simulator...")
	simulator.Stop()
	log.Println("Simulator stopped")
}
