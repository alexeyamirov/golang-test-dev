// Пакет main - точка входа для API Gateway сервиса
// Предоставляет HTTP и gRPC API для метрик и алертов
package main

import (
	"context"   // Контекст для отмены операций и таймаутов
	"fmt"       // Форматирование строк
	"log"       // Логирование
	"net"       // Сетевой listener для gRPC
	"net/http"  // HTTP сервер и клиент
	"os"        // Переменные окружения, выход из программы
	"os/signal" // Обработка сигналов ОС (Ctrl+C)
	"strconv"   // Преобразование строк в числа
	"syscall"   // Системные вызовы (SIGINT, SIGTERM)
	"time"      // Работа со временем

	"github.com/gin-gonic/gin"           // HTTP веб-фреймворк
	tr181pb "golang-test-dev/api/tr181pb/api/proto" // Сгенерированный gRPC код
	"golang-test-dev/pkg/database"      // PostgreSQL и Redis
	"golang-test-dev/pkg/tr181"         // Модель данных TR181
	"google.golang.org/grpc"             // gRPC сервер
	"google.golang.org/grpc/reflection"  // Рефлексия для grpcurl
)

// apiServer - реализует gRPC интерфейс TR181ApiServer
type apiServer struct {
	tr181pb.UnimplementedTR181ApiServer // Встраиваем для обратной совместимости
	postgresDB  *database.PostgresDB   // Подключение к PostgreSQL
	redisCache  *database.RedisCache   // Подключение к Redis для кэша
}

// GetMetric - gRPC метод получения метрик по устройству и периоду
func (s *apiServer) GetMetric(ctx context.Context, req *tr181pb.MetricRequest) (*tr181pb.MetricResponse, error) {
	// Проверяем обязательный параметр
	if req.SerialNumber == "" {
		return nil, fmt.Errorf("serial_number is required")
	}
	// Проверяем валидность типа метрики
	if !isValidMetricType(tr181.MetricType(req.MetricType)) {
		return nil, fmt.Errorf("invalid metric type")
	}

	// Преобразуем Unix timestamp в time.Time для начала периода
	from := time.Unix(req.From, 0)
	// Если не указан - берём последние 24 часа
	if req.From == 0 {
		from = time.Now().Add(-24 * time.Hour)
	}
	// Преобразуем конец периода
	to := time.Unix(req.To, 0)
	// Если не указан - текущее время
	if req.To == 0 {
		to = time.Now()
	}

	// Формируем ключ кэша из параметров запроса
	cacheKey := fmt.Sprintf("metric:%s:%s:%d:%d", req.MetricType, req.SerialNumber, from.Unix(), to.Unix())
	// Пробуем получить данные из Redis кэша
	if cached, err := s.redisCache.GetCachedMetrics(ctx, cacheKey); err == nil && cached != nil {
		// Конвертируем в gRPC формат и возвращаем
		metrics := make([]*tr181pb.MetricValue, len(cached))
		for i, m := range cached {
			metrics[i] = &tr181pb.MetricValue{Value: int32(m.Value), Time: m.Time}
		}
		return &tr181pb.MetricResponse{Metrics: metrics}, nil
	}

	// Запрашиваем метрики из PostgreSQL
	metrics, err := s.postgresDB.GetMetrics(ctx, req.SerialNumber, req.MetricType, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	// Сохраняем результат в кэш на 30 секунд
	s.redisCache.CacheMetrics(ctx, cacheKey, metrics, 30*time.Second)

	// Конвертируем в gRPC формат
	pbMetrics := make([]*tr181pb.MetricValue, len(metrics))
	for i, m := range metrics {
		pbMetrics[i] = &tr181pb.MetricValue{Value: int32(m.Value), Time: m.Time}
	}
	return &tr181pb.MetricResponse{Metrics: pbMetrics}, nil
}

// GetAlert - gRPC метод получения статистики по алертам
func (s *apiServer) GetAlert(ctx context.Context, req *tr181pb.AlertRequest) (*tr181pb.AlertResponse, error) {
	// Проверяем обязательный параметр
	if req.SerialNumber == "" {
		return nil, fmt.Errorf("serial_number is required")
	}
	// Проверяем валидность типа алерта
	if !isValidAlertType(tr181.AlertType(req.AlertType)) {
		return nil, fmt.Errorf("invalid alert type")
	}

	// Парсим период времени
	from := time.Unix(req.From, 0)
	if req.From == 0 {
		from = time.Now().Add(-24 * time.Hour)
	}
	to := time.Unix(req.To, 0)
	if req.To == 0 {
		to = time.Now()
	}

	// Пробуем получить из кэша
	cacheKey := fmt.Sprintf("alert:%s:%s:%d:%d", req.AlertType, req.SerialNumber, from.Unix(), to.Unix())
	if cached, err := s.redisCache.GetCachedAlertStats(ctx, cacheKey); err == nil && cached != nil {
		return &tr181pb.AlertResponse{Value: int32(cached.Value), Count: int32(cached.Count)}, nil
	}

	// Запрашиваем из БД
	stats, err := s.postgresDB.GetAlertStats(ctx, req.SerialNumber, req.AlertType, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert stats: %w", err)
	}

	// Кэшируем результат
	s.redisCache.CacheAlertStats(ctx, cacheKey, stats, 30*time.Second)
	return &tr181pb.AlertResponse{Value: int32(stats.Value), Count: int32(stats.Count)}, nil
}

func main() {
	// Читаем строку подключения к PostgreSQL из переменной окружения
	postgresConnStr := os.Getenv("POSTGRES_CONN_STR")
	// Значение по умолчанию если не задана
	if postgresConnStr == "" {
		postgresConnStr = "postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
	}

	// Подключаемся к PostgreSQL
	postgresDB, err := database.NewPostgresDB(postgresConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	// Закрываем соединение при выходе
	defer postgresDB.Close()

	// Контекст для инициализации
	ctx := context.Background()
	// Создаём таблицы в БД (если ещё не созданы)
	if err := postgresDB.InitSchema(ctx); err != nil {
		log.Printf("Warning: Schema initialization failed (might be OK if TimescaleDB not available): %v", err)
	}

	// Читаем адрес Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Подключаемся к Redis для кэширования
	redisCache, err := database.NewRedisCache(redisAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Настраиваем Gin в release режиме (без отладочной информации)
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	// Создаём HTTP роутер
	router := gin.Default()
	// Добавляем middleware для CPU нагрузки (для автоскейлинга)
	router.Use(cpuLoadMiddleware())

	// Группа роутов с префиксом /api/v1
	api := router.Group("/api/v1")
	{
		// GET /api/v1/metric/:metricType - получение метрик
		api.GET("/metric/:metricType", getMetricHandler(postgresDB, redisCache))
		// GET /api/v1/alert/:alertType - получение статистики алертов
		api.GET("/alert/:alertType", getAlertHandler(postgresDB, redisCache))
	}

	// Health check - проверка работоспособности
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Порт для HTTP (по умолчанию 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Порт для gRPC (по умолчанию 9090)
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	// Создаём HTTP сервер
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Запускаем HTTP сервер в отдельной горутине
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Создаём TCP listener для gRPC на отдельном порту
	grpcListener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	// Создаём gRPC сервер
	grpcServer := grpc.NewServer()
	// Регистрируем наш сервис
	tr181pb.RegisterTR181ApiServer(grpcServer, &apiServer{
		postgresDB: postgresDB,
		redisCache: redisCache,
	})
	// Включаем рефлексию для grpcurl
	reflection.Register(grpcServer)

	// Запускаем gRPC сервер в отдельной горутине
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	// Логируем успешный запуск
	log.Printf("API Gateway started: HTTP on port %s, gRPC on port %s", port, grpcPort)

	// Канал для получения сигналов завершения
	quit := make(chan os.Signal, 1)
	// Регистрируем обработку SIGINT (Ctrl+C) и SIGTERM
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Блокируем до получения сигнала
	<-quit

	// Начинаем корректное завершение
	log.Println("Shutting down server...")
	// Контекст с таймаутом 5 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Останавливаем gRPC сервер
	grpcServer.GracefulStop()
	// Останавливаем HTTP сервер
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

// getMetricHandler - HTTP обработчик для получения метрик
func getMetricHandler(postgresDB *database.PostgresDB, redisCache *database.RedisCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем параметры из URL
		metricType := c.Param("metricType")
		serialNumber := c.Query("serial-number")
		fromStr := c.Query("from")
		toStr := c.Query("to")

		// Проверка обязательного параметра
		if serialNumber == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial-number is required"})
			return
		}

		// Парсим время начала периода
		from, err := parseTime(fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from parameter"})
			return
		}

		// Парсим время конца периода
		to, err := parseTime(toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to parameter"})
			return
		}

		// Проверяем валидность типа метрики
		if !isValidMetricType(tr181.MetricType(metricType)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
			return
		}

		// Формируем ключ кэша
		cacheKey := fmt.Sprintf("metric:%s:%s:%d:%d", metricType, serialNumber, from.Unix(), to.Unix())
		ctx := c.Request.Context()

		// Пробуем получить из кэша
		if cached, err := redisCache.GetCachedMetrics(ctx, cacheKey); err == nil && cached != nil {
			c.JSON(http.StatusOK, cached)
			return
		}

		// Запрашиваем из PostgreSQL
		metrics, err := postgresDB.GetMetrics(ctx, serialNumber, metricType, from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get metrics"})
			return
		}

		// Сохраняем в кэш
		redisCache.CacheMetrics(ctx, cacheKey, metrics, 30*time.Second)
		// Возвращаем результат
		c.JSON(http.StatusOK, metrics)
	}
}

// getAlertHandler - HTTP обработчик для получения статистики алертов
func getAlertHandler(postgresDB *database.PostgresDB, redisCache *database.RedisCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем параметры
		alertType := c.Param("alertType")
		serialNumber := c.Query("serial-number")
		fromStr := c.Query("from")
		toStr := c.Query("to")

		if serialNumber == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "serial-number is required"})
			return
		}

		from, err := parseTime(fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from parameter"})
			return
		}

		to, err := parseTime(toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to parameter"})
			return
		}

		if !isValidAlertType(tr181.AlertType(alertType)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert type"})
			return
		}

		cacheKey := fmt.Sprintf("alert:%s:%s:%d:%d", alertType, serialNumber, from.Unix(), to.Unix())
		ctx := c.Request.Context()

		if cached, err := redisCache.GetCachedAlertStats(ctx, cacheKey); err == nil && cached != nil {
			c.JSON(http.StatusOK, cached)
			return
		}

		stats, err := postgresDB.GetAlertStats(ctx, serialNumber, alertType, from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get alert stats"})
			return
		}

		redisCache.CacheAlertStats(ctx, cacheKey, stats, 30*time.Second)
		c.JSON(http.StatusOK, stats)
	}
}

// parseTime - парсит строку времени в time.Time
func parseTime(timeStr string) (time.Time, error) {
	// Пустая строка = последние 24 часа
	if timeStr == "" {
		return time.Now().Add(-24 * time.Hour), nil
	}

	// Поддерживаемые форматы
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	}

	// Пробуем каждый формат
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	// Пробуем Unix timestamp
	if unixTime, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		return time.Unix(unixTime, 0), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format")
}

// isValidMetricType - проверяет допустимость типа метрики
func isValidMetricType(mt tr181.MetricType) bool {
	validTypes := []tr181.MetricType{
		tr181.MetricCPUUsage, tr181.MetricMemoryUsage, tr181.MetricCPUTemperature,
		tr181.MetricBoardTemperature, tr181.MetricRadioTemperature,
		tr181.MetricWiFi2GHzSignal, tr181.MetricWiFi5GHzSignal, tr181.MetricWiFi6GHzSignal,
		tr181.MetricEthernetBytesSent, tr181.MetricEthernetBytesRecv, tr181.MetricUptime,
	}
	for _, vt := range validTypes {
		if mt == vt {
			return true
		}
	}
	return false
}

// isValidAlertType - проверяет допустимость типа алерта
func isValidAlertType(at tr181.AlertType) bool {
	return at == tr181.AlertHighCPUUsage || at == tr181.AlertLowWiFi
}

// cpuLoadMiddleware - создаёт небольшую CPU нагрузку (для автоскейлинга в Kubernetes)
func cpuLoadMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sum := 0
		// Небольшие вычисления для имитации нагрузки
		for i := 0; i < 1000; i++ {
			sum += i * i
		}
		_ = sum
		c.Next()
	}
}
