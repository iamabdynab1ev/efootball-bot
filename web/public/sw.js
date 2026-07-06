// Service worker eFootLeague — приём web-push и клики по уведомлениям.

self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (e) => e.waitUntil(self.clients.claim()));

self.addEventListener("push", (event) => {
  let data = { title: "eFootLeague", body: "", url: "/", kind: "system" };
  try {
    if (event.data) data = { ...data, ...event.data.json() };
  } catch (e) {
    if (event.data) data.body = event.data.text();
  }
  const kind = data.kind || "system";
  // Вызов на матч — «важное» уведомление: своя вибрация, не схлопывается с
  // другими типами и не исчезает само, пока человек не отреагирует.
  const isChallenge = kind === "challenge";
  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: "/icon-192.png",       // большая цветная иконка (кубок)
      badge: "/badge.png",          // монохромный силуэт для статус-бара
      data: { url: data.url || "/" },
      vibrate: isChallenge ? [120, 60, 120, 60, 200] : [80, 40, 80],
      tag: "efl-" + kind,           // схлопывает дубликаты внутри одного типа
      renotify: true,               // повторное событие того же типа снова звучит
      requireInteraction: isChallenge,
      silent: false,                // системный звук уведомления ОС
    })
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url = (event.notification.data && event.notification.data.url) || "/";
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
      for (const c of list) {
        if ("focus" in c) {
          c.navigate(url);
          return c.focus();
        }
      }
      return self.clients.openWindow(url);
    })
  );
});
