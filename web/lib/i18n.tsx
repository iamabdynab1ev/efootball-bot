"use client";

import React, { createContext, useContext, useEffect, useState } from "react";

export type Lang = "ru" | "uz" | "tg";

const T = {
  ru: {
    nav: {
      home: "Главная", leagues: "Лиги", players: "Игроки",
      profile: "Профиль", admin: "Администратор", login: "Войти",
    },
    common: {
      all: "Все", save: "Сохранить", cancel: "Отмена",
      search: "Поиск...", loading: "Загрузка...", error: "Ошибка",
      elo: "ELO", wins: "В", draws: "Н", losses: "П",
      points: "очков", players: "игроков", leagues: "лиг",
      you: "вы", rank: "Игрок", teamPower: "Сила команды",
      active: "активных", inSystem: "в системе",
      doubleRound: "Двойной круг", singleRound: "Один круг",
    },
    auth: {
      loginTitle: "Вход в систему",
      playerTab: "Игрок", adminTab: "Администратор",
      googleLogin: "Войти через Google",
      username: "Имя пользователя", password: "Пароль",
      adminLogin: "Войти как администратор",
      loggingIn: "Вход...", showPassword: "Показать пароль",
      hidePassword: "Скрыть пароль",
    },
    dashboard: {
      title: "Дашборд", subtitle: "Главная",
      tabs: { overview: "Обзор", matches: "Мои матчи", rating: "Рейтинг" },
      topPlayers: "Топ игроков", myLeagues: "Мои лиги",
      noActions: "Нет активных действий",
      noActionsText: "Новые матчи, счета и подтверждения появятся здесь.",
      actionRequired_one: "матч требует действия",
      actionRequired_few: "матча требуют действия",
      actionRequired_many: "матчей требуют действия",
      enterScore: "введите счёт или подтвердите результат",
      goTo: "Перейти",
      awaitConfirm_one: "матч ждёт вашего подтверждения",
      awaitConfirm_few: "матча ждут вашего подтверждения",
      awaitConfirm_many: "матчей ждут вашего подтверждения",
      loginToSeeLeagues: "Войдите чтобы видеть свои лиги",
      notInLeagues: "Не в лигах", joinLeague: "Вступите в лигу.",
      noPlayers: "Нет игроков", noPlayersText: "Рейтинг появится после регистрации.",
      noLeagues: "Лиг пока нет", noLeaguesText: "Администратор создаст их позже.",
      leaguesCount: "активных",
      matchesWaiting: "ждут меня",
    },
    players: {
      title: "Рейтинг", subtitle: "Статистика",
      tabElo: "ELO рейтинг", tabScorers: "Бомбардиры",
      colNum: "#", colPlayer: "Игрок", colTier: "Уровень",
      colPower: "Сила", colElo: "ELO",
      noPlayers: "Игроки ещё не зарегистрированы",
      noPlayersText: "После входа через веб или регистрации в боте игрок появится здесь.",
      noGoals: "Голов пока нет",
      noGoalsText: "Топ бомбардиров появится после подтверждённых матчей.",
      goals: "голов",
      tierGold: "Золото", tierBlue: "Синий", tierPurple: "Фиолет", tierGreen: "Зелёный",
    },
    leagues: {
      title: "Лиги", subtitle: "Турниры",
      noLeagues: "Лиг пока нет", noLeaguesText: "Турниры появятся здесь.",
      joinBtn: "Вступить", joinedBtn: "В лиге", pendingBtn: "Ожидание",
      joined: "Вы подали заявку", joinedApproved: "Вы в лиге",
      joinSuccess: "Заявка отправлена!",
      joinError: "Не удалось подать заявку",
      maxPlayers: "Макс. игроков",
    },
    leagueDetail: {
      standings: "Таблица", schedule: "Расписание", myMatches: "Мои матчи",
      pos: "М", team: "Команда", pts: "О", w: "В", d: "Н", l: "П",
      gf: "ГЗ", ga: "ГП", gd: "РМ",
      round: "Тур",
      noMatches: "Матчей нет", noMatchesText: "Матчи появятся после жеребьёвки.",
      submitResult: "Ввести счёт", confirm: "Подтвердить", dispute: "Оспорить",
      home: "Дома", away: "Гость",
    },
    profile: {
      title: "Личный кабинет",
      editProfile: "Редактировать профиль",
      displayName: "Игровое имя", displayNamePlaceholder: "Ваше имя",
      teamPower: "Общая сила команды", teamPowerPlaceholder: "Например 3150",
      saveProfile: "Сохранить профиль", saving: "Сохранение...",
      saveSuccess: "Профиль сохранён!", saveError: "Не удалось сохранить",
      telegramLinked: "Telegram привязан", notificationsAvailable: "Уведомления доступны",
      linkTelegram: "Привяжите Telegram для уведомлений (код действует 10 мин).",
      getCode: "Получить код", codeCopied: "Код скопирован",
      codeError: "Не удалось получить код",
      myLeagues: "Мои лиги", history: "История матчей",
      notInLeague: "Вы еще не в лиге", notInLeagueText: "Активных участий пока нет.",
      noHistory: "Истории пока нет", noHistoryText: "Когда гость подтвердит счет, матч появится здесь.",
      openLeagues: "Открыть лиги",
      wins: "Победы", draws: "Ничьи", losses: "Поражения",
      leaguesLabel: "Лиги", pointsLabel: "Очки",
      confirmed: "подтвержден", round: "Тур",
    },
    admin: {
      title: "Администратор", subtitle: "Панель управления",
      tabLeagues: "Лиги", tabRequests: "Заявки",
      tabDisputes: "Споры", tabUsers: "Пользователи",
      createLeague: "Создать лигу", leagueName: "Название лиги",
      create: "Создать", creating: "Создание...",
      archive: "Архивировать", draw: "Жеребьёвка",
      pending: "Ожидают", approved: "Одобрены",
      approve: "Одобрить", reject: "Отклонить",
      resolve: "Разрешить спор", noDisputes: "Нет споров",
      noDisputesText: "Оспариваемые матчи появятся здесь.",
      resetRatings: "Сбросить ELO", resetConfirm: "Сбросить весь ELO рейтинг? Это необратимо.",
      admins: "Администраторы", makeAdmin: "Назначить Админом",
      makeSuperAdmin: "Повысить до Супер-Админа", removeRole: "Снять роль",
      superAdminOnly: "Только для Супер-Администратора",
      drawSuccess: "Расписание сгенерировано!", drawError: "Ошибка жеребьёвки",
      archiveSuccess: "Архивировано", archiveError: "Ошибка архивации",
      noLeagues: "Лиг нет", noMembers: "Участников нет",
      score: "Счёт", homeGoals: "Голы хозяев", awayGoals: "Голы гостей",
      note: "Примечание (необяз.)", resolveBtn: "Применить",
    },
    matchCard: {
      enterScore: "Введите счёт",
      homeGoals: "Голы хозяев", awayGoals: "Голы гостей",
      submit: "Отправить", submitting: "Отправка...",
      confirm: "Подтвердить результат", confirming: "Подтверждение...",
      dispute: "Оспорить", disputing: "Оспаривание...",
      submitSuccess: "Счёт отправлен!", submitError: "Ошибка отправки",
      confirmSuccess: "Подтверждено!", confirmError: "Ошибка подтверждения",
      disputeSuccess: "Спор открыт!", disputeError: "Ошибка",
      pending: "Ожидает подтверждения", scheduled: "Запланирован", disputed: "Оспаривается",
      round: "Тур",
    },
    status: {
      registration: "Регистрация", active: "Активна",
      finished: "Завершена", archived: "Архив", unknown: "Неизвестно",
    },
  },

  uz: {
    nav: {
      home: "Bosh sahifa", leagues: "Ligalar", players: "O'yinchilar",
      profile: "Profil", admin: "Administrator", login: "Kirish",
    },
    common: {
      all: "Barchasi", save: "Saqlash", cancel: "Bekor qilish",
      search: "Qidirish...", loading: "Yuklanmoqda...", error: "Xato",
      elo: "ELO", wins: "G", draws: "D", losses: "Y",
      points: "ball", players: "o'yinchi", leagues: "liga",
      you: "siz", rank: "O'yinchi", teamPower: "Jamoa kuchi",
      active: "faol", inSystem: "tizimda",
      doubleRound: "Ikki davra", singleRound: "Bir davra",
    },
    auth: {
      loginTitle: "Tizimga kirish",
      playerTab: "O'yinchi", adminTab: "Administrator",
      googleLogin: "Google orqali kirish",
      username: "Foydalanuvchi nomi", password: "Parol",
      adminLogin: "Administrator sifatida kirish",
      loggingIn: "Kirish...", showPassword: "Parolni ko'rsatish",
      hidePassword: "Parolni yashirish",
    },
    dashboard: {
      title: "Boshqaruv", subtitle: "Bosh sahifa",
      tabs: { overview: "Umumiy", matches: "Mening o'yinlarim", rating: "Reyting" },
      topPlayers: "Top o'yinchilar", myLeagues: "Mening ligalarim",
      noActions: "Faol harakatlar yo'q",
      noActionsText: "Yangi o'yinlar, hisoblar va tasdiqlashlar bu yerda ko'rinadi.",
      actionRequired_one: "o'yin harakat talab qiladi",
      actionRequired_few: "ta o'yin harakat talab qiladi",
      actionRequired_many: "ta o'yin harakat talab qiladi",
      enterScore: "hisobni kiriting yoki natijani tasdiqlang",
      goTo: "O'tish",
      awaitConfirm_one: "o'yin tasdiqlashingizni kutmoqda",
      awaitConfirm_few: "ta o'yin tasdiqlashingizni kutmoqda",
      awaitConfirm_many: "ta o'yin tasdiqlashingizni kutmoqda",
      loginToSeeLeagues: "Ligalarni ko'rish uchun tizimga kiring",
      notInLeagues: "Ligada emas", joinLeague: "Ligaga qo'shiling.",
      noPlayers: "O'yinchilar yo'q", noPlayersText: "Ro'yxatdan o'tgandan so'ng reyting paydo bo'ladi.",
      noLeagues: "Ligalar yo'q", noLeaguesText: "Administrator ularni keyinroq yaratadi.",
      leaguesCount: "faol", matchesWaiting: "meni kutmoqda",
    },
    players: {
      title: "Reyting", subtitle: "Statistika",
      tabElo: "ELO reytingi", tabScorers: "Bombardirlar",
      colNum: "#", colPlayer: "O'yinchi", colTier: "Daraja",
      colPower: "Kuch", colElo: "ELO",
      noPlayers: "O'yinchilar hali ro'yxatdan o'tmagan",
      noPlayersText: "Veb orqali yoki bot orqali ro'yxatdan o'tgandan so'ng o'yinchi bu yerda ko'rinadi.",
      noGoals: "Hozircha gol yo'q",
      noGoalsText: "Top bombardirlar tasdiqlangan o'yinlardan keyin ko'rinadi.",
      goals: "gol",
      tierGold: "Oltin", tierBlue: "Ko'k", tierPurple: "Binafsha", tierGreen: "Yashil",
    },
    leagues: {
      title: "Ligalar", subtitle: "Turnirlar",
      noLeagues: "Ligalar yo'q", noLeaguesText: "Turnirlar bu yerda ko'rinadi.",
      joinBtn: "Qo'shilish", joinedBtn: "Ligada", pendingBtn: "Kutish",
      joined: "Ariza yubordingiz", joinedApproved: "Siz ligadasiz",
      joinSuccess: "Ariza yuborildi!",
      joinError: "Arizani yuborib bo'lmadi",
      maxPlayers: "Maks. o'yinchilar",
    },
    leagueDetail: {
      standings: "Jadval", schedule: "Jadval", myMatches: "Mening o'yinlarim",
      pos: "O'", team: "Jamoa", pts: "B", w: "G", d: "D", l: "Y",
      gf: "GU", ga: "GK", gd: "GF",
      round: "Tur",
      noMatches: "O'yinlar yo'q", noMatchesText: "O'yinlar qur'a tashlanganidan so'ng paydo bo'ladi.",
      submitResult: "Hisobni kiritish", confirm: "Tasdiqlash", dispute: "E'tiroz",
      home: "Uy", away: "Mehmon",
    },
    profile: {
      title: "Shaxsiy kabinet",
      editProfile: "Profilni tahrirlash",
      displayName: "O'yin nomi", displayNamePlaceholder: "Ismingiz",
      teamPower: "Umumiy jamoa kuchi", teamPowerPlaceholder: "Masalan 3150",
      saveProfile: "Profilni saqlash", saving: "Saqlanmoqda...",
      saveSuccess: "Profil saqlandi!", saveError: "Saqlashda xato",
      telegramLinked: "Telegram ulangan", notificationsAvailable: "Bildirishnomalar mavjud",
      linkTelegram: "Bildirishnomalar uchun Telegramni ulang (kod 10 daqiqa amal qiladi).",
      getCode: "Kod olish", codeCopied: "Kod nusxalandi",
      codeError: "Kodni olishda xato",
      myLeagues: "Mening ligalarim", history: "O'yinlar tarixi",
      notInLeague: "Siz hali ligada emassiz", notInLeagueText: "Faol ishtiroklar yo'q.",
      noHistory: "Tarix yo'q", noHistoryText: "Mehmon hisobni tasdiqlagan vaqtda o'yin bu yerda ko'rinadi.",
      openLeagues: "Ligalarni ochish",
      wins: "G'alabalar", draws: "Durranglar", losses: "Mag'lubiyatlar",
      leaguesLabel: "Ligalar", pointsLabel: "Ballar",
      confirmed: "tasdiqlangan", round: "Tur",
    },
    admin: {
      title: "Administrator", subtitle: "Boshqaruv paneli",
      tabLeagues: "Ligalar", tabRequests: "Arizalar",
      tabDisputes: "Nizolar", tabUsers: "Foydalanuvchilar",
      createLeague: "Liga yaratish", leagueName: "Liga nomi",
      create: "Yaratish", creating: "Yaratilmoqda...",
      archive: "Arxivlash", draw: "Qur'a",
      pending: "Kutmoqda", approved: "Tasdiqlangan",
      approve: "Tasdiqlash", reject: "Rad etish",
      resolve: "Nizoni hal qilish", noDisputes: "Nizolar yo'q",
      noDisputesText: "Nizoli o'yinlar bu yerda ko'rinadi.",
      resetRatings: "ELO ni tiklash", resetConfirm: "Barcha ELO reytingini tiklash? Bu qaytarilmaydi.",
      admins: "Administratorlar", makeAdmin: "Admin tayinlash",
      makeSuperAdmin: "Super-Adminga ko'tarish", removeRole: "Rolni olib tashlash",
      superAdminOnly: "Faqat Super-Administrator uchun",
      drawSuccess: "Jadval yaratildi!", drawError: "Qur'a xatosi",
      archiveSuccess: "Arxivlandi", archiveError: "Arxivlash xatosi",
      noLeagues: "Ligalar yo'q", noMembers: "A'zolar yo'q",
      score: "Hisob", homeGoals: "Uy gollar", awayGoals: "Mehmon gollar",
      note: "Izoh (ixtiyoriy)", resolveBtn: "Qo'llash",
    },
    matchCard: {
      enterScore: "Hisobni kiriting",
      homeGoals: "Uy gollar", awayGoals: "Mehmon gollar",
      submit: "Yuborish", submitting: "Yuborilmoqda...",
      confirm: "Natijani tasdiqlash", confirming: "Tasdiqlanmoqda...",
      dispute: "E'tiroz bildirish", disputing: "E'tiroz...",
      submitSuccess: "Hisob yuborildi!", submitError: "Yuborishda xato",
      confirmSuccess: "Tasdiqlandi!", confirmError: "Tasdiqlanmadi",
      disputeSuccess: "Nizo ochildi!", disputeError: "Xato",
      pending: "Tasdiq kutilmoqda", scheduled: "Rejalashtirilgan", disputed: "Nizoli",
      round: "Tur",
    },
    status: {
      registration: "Ro'yxatga olish", active: "Faol",
      finished: "Tugallangan", archived: "Arxiv", unknown: "Noma'lum",
    },
  },

  tg: {
    nav: {
      home: "Саҳифаи асосӣ", leagues: "Лигаҳо", players: "Бозигарон",
      profile: "Профил", admin: "Маъмур", login: "Даромадан",
    },
    common: {
      all: "Ҳама", save: "Захира кардан", cancel: "Бекор кардан",
      search: "Ҷустуҷӯ...", loading: "Боргирӣ...", error: "Хато",
      elo: "ELO", wins: "Б", draws: "М", losses: "Б",
      points: "хол", players: "бозигар", leagues: "лига",
      you: "шумо", rank: "Бозигар", teamPower: "Қудрати дастa",
      active: "фаъол", inSystem: "дар система",
      doubleRound: "Давраи дугона", singleRound: "Давраи якка",
    },
    auth: {
      loginTitle: "Вуруд ба система",
      playerTab: "Бозигар", adminTab: "Маъмур",
      googleLogin: "Тавассути Google даромадан",
      username: "Номи корбар", password: "Рамз",
      adminLogin: "Ҳамчун маъмур даромадан",
      loggingIn: "Вуруд...", showPassword: "Нишон додани рамз",
      hidePassword: "Пинҳон кардани рамз",
    },
    dashboard: {
      title: "Панели идоракунӣ", subtitle: "Саҳифаи асосӣ",
      tabs: { overview: "Умумӣ", matches: "Бозиҳои ман", rating: "Рейтинг" },
      topPlayers: "Беҳтарин бозигарон", myLeagues: "Лигаҳои ман",
      noActions: "Амалҳои фаъол нест",
      noActionsText: "Бозиҳои нав, натиҷаҳо ва тасдиқҳо дар ин ҷо нишон дода мешаванд.",
      actionRequired_one: "бозӣ амал талаб мекунад",
      actionRequired_few: "бозӣ амал талаб мекунад",
      actionRequired_many: "бозӣ амал талаб мекунад",
      enterScore: "натиҷаро ворид кунед ё тасдиқ кунед",
      goTo: "Гузаштан",
      awaitConfirm_one: "бозӣ тасдиқи шуморо интизор аст",
      awaitConfirm_few: "бозӣ тасдиқи шуморо интизор аст",
      awaitConfirm_many: "бозӣ тасдиқи шуморо интизор аст",
      loginToSeeLeagues: "Барои дидани лигаҳо ворид шавед",
      notInLeagues: "Дар лига нест", joinLeague: "Ба лига ҳамроҳ шавед.",
      noPlayers: "Бозигарон нест", noPlayersText: "Рейтинг пас аз бақайдгирӣ пайдо мешавад.",
      noLeagues: "Лигаҳо нест", noLeaguesText: "Маъмур баъдтар онҳоро эҷод мекунад.",
      leaguesCount: "фаъол", matchesWaiting: "маро интизор аст",
    },
    players: {
      title: "Рейтинг", subtitle: "Омор",
      tabElo: "Рейтинги ELO", tabScorers: "Бомбардирҳо",
      colNum: "#", colPlayer: "Бозигар", colTier: "Сатҳ",
      colPower: "Қудрат", colElo: "ELO",
      noPlayers: "Бозигарон ҳанӯз бақайд нашудаанд",
      noPlayersText: "Пас аз бақайдгирӣ тавассути веб ё бот бозигар дар ин ҷо намоиш дода мешавад.",
      noGoals: "Ҳоло гол нест",
      noGoalsText: "Беҳтарин бомбардирҳо пас аз бозиҳои тасдиқшуда намоиш дода мешаванд.",
      goals: "гол",
      tierGold: "Тилло", tierBlue: "Кабуд", tierPurple: "Бунафш", tierGreen: "Сабз",
    },
    leagues: {
      title: "Лигаҳо", subtitle: "Мусобиқаҳо",
      noLeagues: "Лигаҳо нест", noLeaguesText: "Турнирҳо дар ин ҷо намоиш дода мешаванд.",
      joinBtn: "Ҳамроҳ шудан", joinedBtn: "Дар лига", pendingBtn: "Интизор",
      joined: "Аризаи шумо фиристода шуд", joinedApproved: "Шумо дар лига ҳастед",
      joinSuccess: "Ариза фиристода шуд!",
      joinError: "Аризаро фиристода нашуд",
      maxPlayers: "Ҳадди аксар бозигарон",
    },
    leagueDetail: {
      standings: "Ҷадвал", schedule: "Барнома", myMatches: "Бозиҳои ман",
      pos: "Ҷ", team: "Дастa", pts: "Х", w: "Б", d: "М", l: "Б",
      gf: "ГЗ", ga: "ГК", gd: "ФГ",
      round: "Тур",
      noMatches: "Бозиҳо нест", noMatchesText: "Бозиҳо пас аз қуръакашӣ пайдо мешаванд.",
      submitResult: "Натиҷаро ворид кунед", confirm: "Тасдиқ кардан", dispute: "Эътироз",
      home: "Хона", away: "Меҳмон",
    },
    profile: {
      title: "Кабинети шахсӣ",
      editProfile: "Таҳрири профил",
      displayName: "Номи бозӣ", displayNamePlaceholder: "Номи шумо",
      teamPower: "Қудрати умумии даста", teamPowerPlaceholder: "Масалан 3150",
      saveProfile: "Захира кардани профил", saving: "Захира мешавад...",
      saveSuccess: "Профил захира шуд!", saveError: "Захира нашуд",
      telegramLinked: "Telegram пайваст аст", notificationsAvailable: "Огоҳӣ дастрас аст",
      linkTelegram: "Барои огоҳиҳо Telegramро пайваст кунед (рамз 10 дақиқа амал мекунад).",
      getCode: "Гирифтани рамз", codeCopied: "Рамз нусхабардорӣ шуд",
      codeError: "Рамзро гирифта нашуд",
      myLeagues: "Лигаҳои ман", history: "Таърихи бозиҳо",
      notInLeague: "Шумо ҳанӯз дар лига нестед", notInLeagueText: "Иштирокҳои фаъол нест.",
      noHistory: "Таърих нест", noHistoryText: "Вақте меҳмон натиҷаро тасдиқ мекунад, бозӣ дар ин ҷо намоиш дода мешавад.",
      openLeagues: "Кушодани лигаҳо",
      wins: "Ғалабаҳо", draws: "Мусовӣ", losses: "Мағлубиятҳо",
      leaguesLabel: "Лигаҳо", pointsLabel: "Холҳо",
      confirmed: "тасдиқшуда", round: "Тур",
    },
    admin: {
      title: "Маъмур", subtitle: "Панели идоракунӣ",
      tabLeagues: "Лигаҳо", tabRequests: "Аризаҳо",
      tabDisputes: "Баҳсҳо", tabUsers: "Корбарон",
      createLeague: "Сохтани лига", leagueName: "Номи лига",
      create: "Сохтан", creating: "Сохта мешавад...",
      archive: "Архивкунӣ", draw: "Қуръакашӣ",
      pending: "Интизор", approved: "Тасдиқшуда",
      approve: "Тасдиқ", reject: "Рад кардан",
      resolve: "Ҳал кардани баҳс", noDisputes: "Баҳсҳо нест",
      noDisputesText: "Бозиҳои баҳсноқ дар ин ҷо намоиш дода мешаванд.",
      resetRatings: "Барқарор кардани ELO", resetConfirm: "Тамоми рейтинги ELO барқарор карда шавад? Ин бозгашт надорад.",
      admins: "Маъмурон", makeAdmin: "Маъмур таъин кардан",
      makeSuperAdmin: "Ба Супер-Маъмур баланд бардоштан", removeRole: "Нақшро хориҷ кардан",
      superAdminOnly: "Танҳо барои Супер-Маъмур",
      drawSuccess: "Барнома тартиб дода шуд!", drawError: "Хатои қуръакашӣ",
      archiveSuccess: "Архившуд", archiveError: "Хатои архивкунӣ",
      noLeagues: "Лигаҳо нест", noMembers: "Аъзо нест",
      score: "Натиҷа", homeGoals: "Голҳои хонагӣ", awayGoals: "Голҳои меҳмон",
      note: "Изоҳ (ихтиёрӣ)", resolveBtn: "Татбиқ кардан",
    },
    matchCard: {
      enterScore: "Натиҷаро ворид кунед",
      homeGoals: "Голҳои хонагӣ", awayGoals: "Голҳои меҳмон",
      submit: "Фиристодан", submitting: "Фиристода мешавад...",
      confirm: "Натиҷаро тасдиқ кардан", confirming: "Тасдиқ мешавад...",
      dispute: "Эътироз кардан", disputing: "Эътироз...",
      submitSuccess: "Натиҷа фиристода шуд!", submitError: "Хатои фиристодан",
      confirmSuccess: "Тасдиқшуд!", confirmError: "Тасдиқ нашуд",
      disputeSuccess: "Баҳс кушода шуд!", disputeError: "Хато",
      pending: "Тасдиқро интизор аст", scheduled: "Банақша", disputed: "Баҳсӣ",
      round: "Тур",
    },
    status: {
      registration: "Бақайдгирӣ", active: "Фаъол",
      finished: "Анҷомёфта", archived: "Архив", unknown: "Номаълум",
    },
  },
} as const;

export type Translations = typeof T.ru;
type DeepKeyOf<T> = T extends object
  ? { [K in keyof T]: K extends string ? `${K}` | `${K}.${DeepKeyOf<T[K]>}` : never }[keyof T]
  : never;

type PathValue<T, P extends string> =
  P extends `${infer K}.${infer Rest}`
    ? K extends keyof T ? PathValue<T[K], Rest> : never
    : P extends keyof T ? T[P] : never;

interface LangCtx {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: <P extends DeepKeyOf<Translations>>(path: P) => PathValue<Translations, P>;
}

const LangContext = createContext<LangCtx>({} as LangCtx);

function getPath(obj: any, path: string): any {
  return path.split(".").reduce((acc, k) => acc?.[k], obj);
}

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>("ru");

  useEffect(() => {
    const saved = localStorage.getItem("lang") as Lang | null;
    if (saved && (saved === "ru" || saved === "uz" || saved === "tg")) setLangState(saved);
  }, []);

  const setLang = (l: Lang) => {
    setLangState(l);
    localStorage.setItem("lang", l);
  };

  const t = (path: string) => getPath(T[lang], path) ?? getPath(T.ru, path) ?? path;

  return (
    <LangContext.Provider value={{ lang, setLang, t: t as any }}>
      {children}
    </LangContext.Provider>
  );
}

export function useLang() {
  return useContext(LangContext);
}

export const LANG_LABELS: Record<Lang, string> = {
  ru: "RU", uz: "UZ", tg: "TG",
};
