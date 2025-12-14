package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

// Структура для хранения состояния диалога
type UserState struct {
	ChatID      int64
	ServiceType string            // "water" или "3d"
	Step        int               // текущий шаг в диалоге
	Data        map[string]string // собранные данные
}

// База данных в памяти
var userStates = make(map[int64]*UserState)

func main() {
	// Загрузка токена
	godotenv.Load()
	token := os.Getenv("TELEGRAM_BOT_TOKEN")

	if token == "" {
		log.Panic("❌ ТОКЕН НЕ НАЙДЕН! Создайте файл .env с TELEGRAM_BOT_TOKEN=ваш_токен")
	}

	// 2. Создание бота
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic("❌ Ошибка создания бота: ", err)
	}

	bot.Debug = true
	log.Printf("✅ Бот %s запущен и ждет заказы...", bot.Self.UserName)

	// 3. Настройка обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// 4. Главный цикл
	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		text := update.Message.Text

		log.Printf("[%d] %s", chatID, text)
		handleMessage(bot, chatID, text)
	}
}

// Обработка входящих сообщений
func handleMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	log.Printf("DEBUG: Обрабатываем сообщение '%s' для чата %d", text, chatID)

	state, exists := userStates[chatID]

	// Проверка /start или /restart
	if text == "/start" || text == "/restart" || text == "🔄 Начать заново" {
		sendStartMessage(bot, chatID)
		delete(userStates, chatID) // Сброс
		return
	}

	// Проверка нажатия кнопок услуг
	if strings.Contains(text, "Консультация") && strings.Contains(text, "водоснабжению") {
		log.Printf("DEBUG: Нажата кнопка водоснабжения")
		startWaterConsultation(bot, chatID)
		return
	}

	if strings.Contains(text, "Разработка") && strings.Contains(text, "3D") {
		log.Printf("DEBUG: Нажата кнопка 3D модели")
		start3DModeling(bot, chatID)
		return
	}

	// Проверка нажатия кнопки "Готово"
	if text == "Готово" {
		if exists {
			continueDialog(bot, chatID, text, state)
		} else {
			sendStartMessage(bot, chatID)
		}
		return
	}

	// Если есть активное состояние - продолжаем диалог
	if exists {
		continueDialog(bot, chatID, text, state)
		return
	}

	// Если ничего не подошло - показываем стартовое сообщение
	sendStartMessage(bot, chatID)
}

// Проверка, является ли текст кнопкой услуги
func isServiceButton(text string) bool {
	return strings.Contains(text, "Консультация") ||
		strings.Contains(text, "Разработка") ||
		text == "1" || text == "2"
}

// Приветственное сообщение
func sendStartMessage(bot *tgbotapi.BotAPI, chatID int64) {
	text := `👋 *Здравствуйте! Я бот компании ВиТИМ* (Водоснабжение и Технологии Информационного Моделирования)

*Чем я могу вам помочь?*

Мы специализируемся на:
• 🏗️ Проектировании систем водоснабжения и водоотведения
• 🔧 Разработке 3D моделей на языке GDL для ArchiCAD

*Выберите услугу:*
1️⃣ Консультация по водоснабжению
2️⃣ Разработка 3D модели

_В любой момент можете отправить /restart для начала заново_`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	// Клавиатура
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("1️⃣ Консультация по водоснабжению"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("2️⃣ Разработка 3D модели"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔄 Начать заново"),
		),
	)
	msg.ReplyMarkup = keyboard

	bot.Send(msg)
}

// Начало консультации по водоснабжению
func startWaterConsultation(bot *tgbotapi.BotAPI, chatID int64) {
	userStates[chatID] = &UserState{
		ChatID:      chatID,
		ServiceType: "water",
		Step:        1,
		Data:        make(map[string]string),
	}

	text := `💧 *Консультация по водоснабжению*

*Шаг 1 из 6*
*Проект какой системы необходимо разработать?*
(Например: ХВС, ГВС, канализация, водосток и т.д.)`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Продолжение диалога
func continueDialog(bot *tgbotapi.BotAPI, chatID int64, text string, state *UserState) {
	if state.ServiceType == "water" {
		continueWaterDialog(bot, chatID, text, state)
	} else {
		continue3DDialog(bot, chatID, text, state)
	}
}

// Диалог для водоснабжения
func continueWaterDialog(bot *tgbotapi.BotAPI, chatID int64, text string, state *UserState) {
	// Проверка на restart и пустые сообщения
	if text == "/restart" || text == "🔄 Начать заново" {
		sendStartMessage(bot, chatID)
		delete(userStates, chatID)
		return
	}

	// Защита от пустых сообщений
	if strings.TrimSpace(text) == "" {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Пожалуйста, введите текст. Попробуйте снова:")
		bot.Send(msg)
		return
	}

	switch state.Step {
	case 1: // Система
		state.Data["system_type"] = text
		state.Step = 2

		question := `💧 *Консультация по водоснабжению*

*Шаг 2 из 6*
*Введите наименование объекта:*
(жилой дом / гостиница / школа / больница / другое)

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"
		bot.Send(msg)

	case 2: // Объект
		state.Data["object_type"] = text
		state.Step = 3

		question := `💧 *Консультация по водоснабжению*

*Шаг 3 из 6*
*Введите данные об объекте:*
• Месторасположение
• Этажность
• Строительный объем
        
*Пример:* Москва, ул. Ленина 10, 5 этажей, 12000 м³

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"
		bot.Send(msg)

	case 3: // Данные объекта
		state.Data["object_details"] = text
		state.Step = 4

		question := `💧 *Консультация по водоснабжению*

*Шаг 4 из 6*
*Дополнительная информация:*
1. Получено ли разрешение на строительство?
2. Какой предполагаемый срок проектирования?

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"
		bot.Send(msg)

	case 4: // Доп. информация
		state.Data["additional_info"] = text
		state.Step = 5

		question := `💧 *Консультация по водоснабжению*

*Шаг 5 из 6*
*Введите контактные данные для связи:*
• Email (обязательно)
• Телефон
• Telegram (если отличается от текущего)

*Пример:* client@email.ru, +79161234567, @username

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"
		bot.Send(msg)

	case 5: // Контактные данные
		state.Data["contacts"] = text
		state.Step = 6

		// Извлечение email
		email := "указанный email"
		// Простой поиск email в тексте
		if len(text) > 0 {
			email = text
		}

		question := `💧 *Консультация по водоснабжению*

*Шаг 6 из 6*
*Последний вопрос:*
Спасибо! Вся информация собрана.

✅ *Мы вышлем проект коммерческого предложения на ` + email + ` в течение 2-х часов.*

Для подтверждения отправки заявки нажмите "Готово".

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"

		// Кнопка для подтверждения + кнопка restart
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Готово"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("🔄 Начать заново"),
			),
		)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)

	case 6: // Подтверждение "Готово"
		if text == "Готово" {
			// Сохранение заявки в файл
			saveApplicationToFile(chatID, state.Data, "water")

			// Финальное сообщение
			summary := `✅ *Спасибо за обращение в компанию ВиТИМ!*

✅ *Ваша заявка принята!*

📧 *Мы вышлем проект коммерческого предложения на указанный email в течение 2-х часов.*

👨‍💼 *С вами также свяжется специалист нашей компании в течение часа для уточнения деталей.*

*Собранные данные:*
• 🏗️ Система: ` + state.Data["system_type"] + `
• 🏢 Объект: ` + state.Data["object_type"] + `
• 📍 Детали: ` + state.Data["object_details"] + `
• 📋 Доп. информация: ` + state.Data["additional_info"] + `
• 📞 Контакты: ` + state.Data["contacts"] + `

_Заявка №` + time.Now().Format("2006-01-02_15-04-05")

			msg := tgbotapi.NewMessage(chatID, summary)
			msg.ParseMode = "Markdown"

			// Убираем клавиатуру
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(msg)

			// Отправка уведомления админу
			sendAdminNotification(bot, "💧 НОВАЯ ЗАЯВКА ПО ВОДОСНАБЖЕНИЮ", state.Data)

			// Удаляем состояние
			delete(userStates, chatID)
		} else if text == "/restart" || text == "🔄 Начать заново" {
			sendStartMessage(bot, chatID)
			delete(userStates, chatID)
		} else {
			// Если не "Готово", просим подтвердить
			msg := tgbotapi.NewMessage(chatID, "Для завершения заявки нажмите кнопку 'Готово' или '🔄 Начать заново' для отмены")
			bot.Send(msg)
		}
	}
}

// Начало разработки 3D модели
func start3DModeling(bot *tgbotapi.BotAPI, chatID int64) {
	userStates[chatID] = &UserState{
		ChatID:      chatID,
		ServiceType: "3d",
		Step:        1,
		Data:        make(map[string]string),
	}

	text := `🔄 *Разработка 3D модели для ArchiCAD*

*Шаг 1 из 4*
*Введите название 3D элемента, который вам необходимо разработать:*
(Например: Специальный клапан, Декоративная решетка и т.д.)`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Диалог для 3D моделей
func continue3DDialog(bot *tgbotapi.BotAPI, chatID int64, text string, state *UserState) {
	// Проверка на restart
	if text == "/restart" || text == "🔄 Начать заново" {
		sendStartMessage(bot, chatID)
		delete(userStates, chatID)
		return
	}

	// Защита от пустых сообщений
	if strings.TrimSpace(text) == "" {
		msg := tgbotapi.NewMessage(chatID, "⚠️ Пожалуйста, введите текст. Попробуйте снова:")
		bot.Send(msg)
		return
	}

	switch state.Step {
	case 1: // Название элемента
		state.Data["element_name"] = text
		state.Step = 2

		question := `🔄 *Разработка 3D модели для ArchiCAD*

*Шаг 2 из 4*
*Введите требования для разработки:*
• Примерные габариты
• Примерное количество конфигураций
• Требования к пользовательскому интерфейсу
        
*Пример:* 300x400x500 мм, 3 конфигурации, простой интерфейс с выпадающим списком

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"
		bot.Send(msg)

	case 2: // Требования
		state.Data["requirements"] = text
		state.Step = 3

		question := `🔄 *Разработка 3D модели для ArchiCAD*

*Шаг 3 из 4*
*Введите контактные данные для связи:*
• Email (обязательно)
• Телефон
• Telegram (если отличается от текущего)

*Пример:* designer@studio.ru, +79167654321, @designer

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"
		bot.Send(msg)

	case 3: // Контактные данные
		state.Data["contacts"] = text
		state.Step = 4

		email := "указанный email"
		if len(text) > 0 {
			email = text
		}

		question := `🔄 *Разработка 3D модели для ArchiCAD*

*Шаг 4 из 4*
*Последний вопрос:*
Спасибо! Вся информация собрана.

✅ *Мы вышлем проект коммерческого предложения на ` + email + ` в течение 2-х часов.*

Для подтверждения отправки заявки нажмите "Готово".

_Для отмены отправьте /restart_`

		msg := tgbotapi.NewMessage(chatID, question)
		msg.ParseMode = "Markdown"

		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Готово"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("🔄 Начать заново"),
			),
		)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)

	case 4: // Подтверждение
		if text == "Готово" {
			saveApplicationToFile(chatID, state.Data, "3d")

			summary := `✅ *Спасибо за обращение в компаню ВиТИМ!*

✅ *Ваша заявка принята!*

📧 *Мы вышлем проект коммерческого предложения на указанный email в течение 2-х часов.*

👨‍💻 *С вами также свяжется наш 3D-специалист в течение часа для уточнения технических деталей.*

*Собранные данные:*
• 🔧 Название элемента: ` + state.Data["element_name"] + `
• 📏 Требования: ` + state.Data["requirements"] + `
• 📞 Контакты: ` + state.Data["contacts"] + `

_Заявка №` + time.Now().Format("2006-01-02_15-04-05") //+ `_` //+ fmt.Sprintf("%d", time.Now().Unix()) + `_`

			msg := tgbotapi.NewMessage(chatID, summary)
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(msg)

			sendAdminNotification(bot, "🔄 НОВАЯ ЗАЯВКА НА 3D МОДЕЛЬ", state.Data)

			delete(userStates, chatID)
		} else if text == "/restart" || text == "🔄 Начать заново" {
			sendStartMessage(bot, chatID)
			delete(userStates, chatID)
		} else {
			msg := tgbotapi.NewMessage(chatID, "Для завершения заявки нажмите кнопку 'Готово' или '🔄 Начать заново' для отмены")
			bot.Send(msg)
		}
	}
}

// Сохранение заявки в файл
func saveApplicationToFile(chatID int64, data map[string]string, serviceType string) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("заявки/заявка_%s_%d_%s.txt", serviceType, chatID, timestamp)

	// Создаем папку если нет
	os.MkdirAll("заявки", 0755)

	content := fmt.Sprintf("=== ЗАЯВКА ===\n")
	content += fmt.Sprintf("Тип: %s\n", serviceType)
	content += fmt.Sprintf("ChatID: %d\n", chatID)
	content += fmt.Sprintf("Время: %s\n\n", time.Now().Format("02.01.2006 15:04"))

	for key, value := range data {
		content += fmt.Sprintf("%s: %s\n", key, value)
	}

	content += fmt.Sprintf("\n=== КОНЕЦ ЗАЯВКИ ===\n")

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		log.Printf("❌ Ошибка сохранения файла: %v", err)
	} else {
		log.Printf("✅ Заявка сохранена в файл: %s", filename)
	}
}

// Отправка уведомления админу
func sendAdminNotification(bot *tgbotapi.BotAPI, title string, data map[string]string) {

	adminChatID := int64(7082303368)

	message := fmt.Sprintf("🚨 *%s*\n\n", title)
	message += fmt.Sprintf("📅 *Время:* %s\n\n", time.Now().Format("02.01.2006 15:04"))

	for key, value := range data {
		var fieldName string
		switch key {
		case "system_type":
			fieldName = "🏗️ Система"
		case "object_type":
			fieldName = "🏢 Объект"
		case "object_details":
			fieldName = "📍 Детали объекта"
		case "additional_info":
			fieldName = "📋 Доп. информация"
		case "element_name":
			fieldName = "🔧 Название элемента"
		case "requirements":
			fieldName = "📏 Требования"
		case "contacts":
			fieldName = "📞 Контакты"
		default:
			fieldName = key
		}
		message += fmt.Sprintf("*%s:* %s\n", fieldName, value)
	}

	msg := tgbotapi.NewMessage(adminChatID, message)
	msg.ParseMode = "Markdown"

	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("❌ Ошибка отправки админу: %v", err)
	}
}

