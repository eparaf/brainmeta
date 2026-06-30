/* BrainMeta embeddable widgets — a contact FORM and a step-by-step appointment
 * CALENDAR (AI-recommended slot → service → doctor → month-grid date → time →
 * details). Dark/light themed, fixed-width, configured per clinic via the key.
 *
 *   <script src="https://api.example.com/embed/widget.js"
 *           data-key="pk_xxx" data-mode="calendar"></script>
 *
 * data-mode: "form" | "calendar" | "both"  (default "form")
 * data-api / data-target optional.
 */
(function () {
  "use strict";
  var script = document.currentScript;
  if (!script) {
    var ss = document.getElementsByTagName("script");
    script = ss[ss.length - 1];
  }
  var key = script.getAttribute("data-key");
  var mode = script.getAttribute("data-mode") || "form";
  var api = script.getAttribute("data-api");
  if (!api) {
    try {
      api = new URL(script.src).origin;
    } catch (e) {
      api = "";
    }
  }
  if (!key) {
    console.error("[brainmeta] data-key gerekli");
    return;
  }

  var mount = document.createElement("div");
  var target = script.getAttribute("data-target");
  var hostEl = target && document.querySelector(target);
  if (hostEl) hostEl.appendChild(mount);
  else script.parentNode.insertBefore(mount, script.nextSibling);

  function el(tag, attrs, html) {
    var e = document.createElement(tag);
    attrs = attrs || {};
    for (var k in attrs) {
      if (k === "style") e.style.cssText = attrs[k];
      else e.setAttribute(k, attrs[k]);
    }
    if (html != null) e.innerHTML = html;
    return e;
  }
  function call(path, opts) {
    return fetch(api + path, opts).then(function (r) {
      return r.json();
    });
  }
  function esc(s) {
    return String(s == null ? "" : s).replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }
  function pad(n) {
    return (n < 10 ? "0" : "") + n;
  }
  function fmtDate(d) {
    return d.getFullYear() + "-" + pad(d.getMonth() + 1) + "-" + pad(d.getDate());
  }
  function sameYMD(a, b) {
    return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  }
  function midnight(d) {
    return new Date(d.getFullYear(), d.getMonth(), d.getDate());
  }

  call("/public/widget?key=" + encodeURIComponent(key))
    .then(function (cfg) {
      if (!cfg || cfg.error) {
        mount.textContent = "Widget yüklenemedi (geçersiz anahtar).";
        return;
      }
      render(cfg);
    })
    .catch(function () {
      mount.textContent = "Widget yüklenemedi.";
    });

  function render(cfg) {
    var formColor = cfg.primaryColor || "#30d158";
    var calColor = cfg.calendarColor || formColor;
    var theme = cfg.theme === "light" ? "bm-light" : "bm-dark";
    mount.appendChild(el("style", {}, css()));
    var card = el("div", { class: "bm-w " + theme, style: "--acc:" + calColor + ";--acc-form:" + formColor });
    mount.appendChild(card);
    var showForm = mode === "form" || mode === "both";
    var showCal = mode === "calendar" || mode === "both";
    if (showForm) card.appendChild(formSection(cfg));
    if (showCal) card.appendChild(calendarSection(cfg));
  }

  /* ---- FORM widget --------------------------------------------------------- */
  function formSection(cfg) {
    var sec = el("div", { class: "bm-sec bm-form" });
    sec.appendChild(el("h3", {}, esc(cfg.formTitle || "Randevu Talebi")));
    if (cfg.formSubtitle) sec.appendChild(el("p", { class: "bm-sub" }, esc(cfg.formSubtitle)));
    var inputs = {};
    (cfg.fields || []).forEach(function (f) {
      sec.appendChild(el("label", {}, esc(f.label) + (f.required ? " *" : "")));
      var input =
        f.key === "message"
          ? el("textarea", { rows: "3" })
          : el("input", { type: f.key === "email" ? "email" : f.key === "phone" ? "tel" : "text" });
      inputs[f.key] = input;
      sec.appendChild(input);
    });
    var err = el("div", { class: "bm-err" });
    var btn = el("button", { class: "bm-btn bm-btn-form" }, "Gönder");
    btn.onclick = function () {
      var data = {};
      var missing = false;
      (cfg.fields || []).forEach(function (f) {
        data[f.key] = (inputs[f.key].value || "").trim();
        if (f.required && !data[f.key]) missing = true;
      });
      if (missing) {
        err.textContent = "Lütfen zorunlu alanları doldurun.";
        return;
      }
      err.textContent = "";
      btn.disabled = true;
      btn.textContent = "Gönderiliyor…";
      call("/public/widget/lead?key=" + encodeURIComponent(key), {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify(data),
      })
        .then(function (res) {
          if (res && res.ok) sec.innerHTML = '<div class="bm-ok"><div class="bm-ok-check">✓</div>' + esc(cfg.successText || "Teşekkürler!") + "</div>";
          else fail();
        })
        .catch(fail);
      function fail() {
        err.textContent = "Gönderilemedi, tekrar deneyin.";
        btn.disabled = false;
        btn.textContent = "Gönder";
      }
    };
    sec.appendChild(btn);
    sec.appendChild(err);
    return sec;
  }

  /* ---- CALENDAR widget (multi-step, month grid) --------------------------- */
  function calendarSection(cfg) {
    var WD = ["Pt", "Sa", "Ça", "Pe", "Cu", "Ct", "Pz"];
    var today = new Date();
    var root = el("div", { class: "bm-sec bm-cal" });
    var head = el("div", { class: "bm-cal-head" });
    var body = el("div", { class: "bm-cal-body" });
    root.appendChild(head);
    root.appendChild(body);
    var st = { step: 0, service: null, doctor: null, date: null, slot: null, view: { y: today.getFullYear(), m: today.getMonth() } };

    draw();
    return root;

    function draw() {
      head.innerHTML = "";
      var row = el("div", { class: "bm-cal-titlerow" });
      if (st.step > 0 && st.step < 5) {
        var back = el("button", { class: "bm-back" }, "‹ Geri");
        back.onclick = function () {
          st.step = Math.max(0, st.step - 1);
          draw();
        };
        row.appendChild(back);
      }
      row.appendChild(el("div", { class: "bm-cal-title" }, esc(cfg.calendarTitle || "Online Randevu")));
      head.appendChild(row);
      if (st.step < 5) head.appendChild(stepsBar());
      body.innerHTML = "";
      [stepService, stepDoctor, stepDate, stepSlot, stepDetails][st.step]();
    }
    function stepsBar() {
      var wrap = el("div", { class: "bm-steps" });
      for (var i = 0; i < 5; i++) wrap.appendChild(el("div", { class: "bm-step" + (i === st.step ? " bm-on" : i < st.step ? " bm-done" : "") }));
      return wrap;
    }
    function loading() {
      body.appendChild(el("div", { class: "bm-loading" }, "Yükleniyor…"));
    }
    function empty(t) {
      body.appendChild(el("p", { class: "bm-empty" }, esc(t)));
    }

    function stepService() {
      if (cfg.calendarSubtitle) body.appendChild(el("p", { class: "bm-sub" }, esc(cfg.calendarSubtitle)));
      loading();
      call("/public/widget/services?key=" + encodeURIComponent(key)).then(function (list) {
        body.innerHTML = "";
        if (cfg.calendarSubtitle) body.appendChild(el("p", { class: "bm-sub" }, esc(cfg.calendarSubtitle)));
        if (!list || !list.length) return empty("Şu an online randevuya açık hizmet yok.");
        list.forEach(function (svc) {
          var c = el("button", { class: "bm-row" },
            '<span class="bm-row-name">' + esc(svc.name) + '</span><span class="bm-pill">' + svc.durationMins + ' dk</span><span class="bm-chev">›</span>');
          c.onclick = function () {
            st.service = svc;
            st.doctor = null;
            st.step = 1;
            draw();
          };
          body.appendChild(c);
        });
      });
    }

    function stepDoctor() {
      loading();
      var jobs = [call("/public/widget/doctors?key=" + encodeURIComponent(key) + "&serviceId=" + encodeURIComponent(st.service.id))];
      jobs.push(cfg.recommend ? call("/public/widget/recommend?key=" + encodeURIComponent(key) + "&serviceId=" + encodeURIComponent(st.service.id)).catch(function () { return null; }) : Promise.resolve(null));
      Promise.all(jobs).then(function (res) {
        var list = res[0],
          rec = res[1];
        body.innerHTML = "";
        if (rec && rec.available) {
          var d = rec.doctor;
          var card = el("button", { class: "bm-rec" },
            '<span class="bm-rec-spark">✨</span><span class="bm-col"><span class="bm-rec-top">Önerilen — en erken</span><span class="bm-rec-main">' +
              esc(rec.slot.label) + '</span><span class="bm-rec-sub">' + esc((d.title ? d.title + " " : "") + d.name) + "</span></span><span class='bm-rec-go'>Seç ›</span>");
          card.onclick = function () {
            st.doctor = d;
            st.slot = rec.slot;
            st.date = new Date(rec.slot.iso);
            st.step = 4;
            draw();
          };
          body.appendChild(card);
          body.appendChild(el("div", { class: "bm-or" }, "veya hekim seçin"));
        }
        if (!list || !list.length) return empty("Bu hizmet için uygun hekim yok.");
        list.forEach(function (doc) {
          var initials = (doc.name || "?").split(" ").map(function (p) { return p[0]; }).join("").slice(0, 2).toUpperCase();
          var c = el("button", { class: "bm-row" },
            '<span class="bm-av">' + esc(initials) + '</span><span class="bm-col"><span class="bm-row-name">' +
              esc((doc.title ? doc.title + " " : "") + doc.name) + '</span><span class="bm-row-meta">' + esc(doc.specialty || "") + '</span></span><span class="bm-chev">›</span>');
          c.onclick = function () {
            st.doctor = doc;
            st.date = null;
            st.step = 2;
            draw();
          };
          body.appendChild(c);
        });
      });
    }

    function stepDate() {
      var workdays = st.doctor.days || [1, 2, 3, 4, 5];
      var nav = el("div", { class: "bm-month-nav" });
      var prev = el("button", { class: "bm-nav" }, "‹");
      var next = el("button", { class: "bm-nav" }, "›");
      var lbl = el("div", { class: "bm-month-lbl" }, new Date(st.view.y, st.view.m, 1).toLocaleDateString("tr-TR", { month: "long", year: "numeric" }));
      if (st.view.y === today.getFullYear() && st.view.m === today.getMonth()) prev.disabled = true;
      prev.onclick = function () { shift(-1); };
      next.onclick = function () { shift(1); };
      nav.appendChild(prev);
      nav.appendChild(lbl);
      nav.appendChild(next);
      body.appendChild(nav);

      var gridHead = el("div", { class: "bm-grid bm-grid-h" });
      WD.forEach(function (w) { gridHead.appendChild(el("div", { class: "bm-wd" }, w)); });
      body.appendChild(gridHead);

      var grid = el("div", { class: "bm-grid" });
      var first = new Date(st.view.y, st.view.m, 1);
      var lead = (first.getDay() + 6) % 7;
      var dim = new Date(st.view.y, st.view.m + 1, 0).getDate();
      var t0 = midnight(today);
      for (var i = 0; i < lead; i++) grid.appendChild(el("div", { class: "bm-day bm-empty-day" }, ""));
      for (var day = 1; day <= dim; day++) {
        (function (day) {
          var date = new Date(st.view.y, st.view.m, day);
          var iso = ((date.getDay() + 6) % 7) + 1;
          var ok = date >= t0 && workdays.indexOf(iso) >= 0;
          var cls = "bm-day" + (sameYMD(date, today) ? " bm-today" : "") + (ok ? "" : " bm-off") + (st.date && sameYMD(date, st.date) ? " bm-sel" : "");
          var cell = el("button", { class: cls }, "<span>" + day + "</span>" + (ok ? '<i class="bm-dot"></i>' : ""));
          if (ok)
            cell.onclick = function () {
              st.date = date;
              st.slot = null;
              st.step = 3;
              draw();
            };
          else cell.disabled = true;
          grid.appendChild(cell);
        })(day);
      }
      body.appendChild(grid);
      function shift(d) {
        var m = st.view.m + d,
          y = st.view.y;
        if (m < 0) { m = 11; y--; }
        if (m > 11) { m = 0; y++; }
        st.view = { y: y, m: m };
        draw();
      }
    }

    function stepSlot() {
      var lbl = st.date.toLocaleDateString("tr-TR", { weekday: "long", day: "2-digit", month: "long" });
      body.appendChild(el("p", { class: "bm-sub" }, lbl + " — saat seçin"));
      loading();
      call("/public/widget/availability?key=" + encodeURIComponent(key) + "&doctorId=" + encodeURIComponent(st.doctor.id) + "&date=" + fmtDate(st.date)).then(function (list) {
        body.innerHTML = "";
        body.appendChild(el("p", { class: "bm-sub" }, lbl + " — saat seçin"));
        if (!list || !list.length) return empty("Bu gün için uygun saat yok. Başka gün deneyin.");
        var wrap = el("div", { class: "bm-slots" });
        list.forEach(function (s) {
          var chip = el("button", { class: "bm-slot" }, s.label);
          chip.onclick = function () {
            st.slot = s;
            st.step = 4;
            draw();
          };
          wrap.appendChild(chip);
        });
        body.appendChild(wrap);
      });
    }

    function stepDetails() {
      body.appendChild(el("div", { class: "bm-summary" },
        "<b>" + esc(st.service.name) + "</b> · " + esc((st.doctor.title ? st.doctor.title + " " : "") + st.doctor.name) +
          "<br>" + st.date.toLocaleDateString("tr-TR", { weekday: "long", day: "2-digit", month: "long" }) + " · " + esc(st.slot.label)));
      var name = el("input", { type: "text" });
      var phone = el("input", { type: "tel" });
      var note = el("textarea", { rows: "2" });
      body.appendChild(el("label", {}, "Ad Soyad"));
      body.appendChild(name);
      body.appendChild(el("label", {}, "Telefon"));
      body.appendChild(phone);
      body.appendChild(el("label", {}, "Not (opsiyonel)"));
      body.appendChild(note);
      var err = el("div", { class: "bm-err" });
      var btn = el("button", { class: "bm-btn bm-btn-cal" }, "Randevuyu Onayla");
      btn.onclick = function () {
        if (!name.value.trim() || !phone.value.trim()) {
          err.textContent = "Ad ve telefon gerekli.";
          return;
        }
        err.textContent = "";
        btn.disabled = true;
        btn.textContent = "Gönderiliyor…";
        call("/public/widget/book?key=" + encodeURIComponent(key), {
          method: "POST",
          headers: { "content-type": "application/json" },
          body: JSON.stringify({
            serviceId: st.service.id,
            doctorId: st.doctor.id,
            slot: st.slot.iso,
            name: name.value.trim(),
            phone: phone.value.trim(),
            note: note.value.trim(),
          }),
        })
          .then(function (res) {
            if (res && res.ok) {
              st.step = 5;
              success();
            } else if (res && res.error === "slot_taken") {
              err.textContent = "Bu saat az önce doldu, lütfen başka saat seçin.";
              reset();
            } else {
              err.textContent = "Randevu oluşturulamadı.";
              reset();
            }
          })
          .catch(function () {
            err.textContent = "Bağlantı hatası.";
            reset();
          });
        function reset() {
          btn.disabled = false;
          btn.textContent = "Randevuyu Onayla";
        }
      };
      body.appendChild(btn);
      body.appendChild(err);
    }

    function success() {
      head.innerHTML = "";
      body.innerHTML = "";
      body.appendChild(el("div", { class: "bm-ok" },
        '<div class="bm-ok-check">✓</div>' + esc(cfg.confirmText || "Randevu talebiniz alındı!") +
          '<div class="bm-ok-when">' + st.date.toLocaleDateString("tr-TR", { weekday: "long", day: "2-digit", month: "long" }) +
          " · " + esc(st.slot.label) + "</div>"));
    }
  }

  function css() {
    return [
      "@keyframes bm-in{from{opacity:0;transform:translateY(6px)}to{opacity:1;transform:none}}",
      ".bm-w{width:380px;max-width:calc(100vw - 24px);box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,'SF Pro Display','SF Pro Text','Inter',system-ui,sans-serif;-webkit-font-smoothing:antialiased;border-radius:24px;padding:22px;border:.5px solid var(--bd);background:var(--bg);color:var(--text);box-shadow:0 1px 2px rgba(0,0,0,.18),0 18px 50px rgba(0,0,0,.30)}",
      ".bm-w.bm-dark{--bg:#1c1c1e;--panel:#2c2c2e;--text:#f5f5f7;--muted:#98989d;--bd:rgba(255,255,255,.10);--hover:rgba(255,255,255,.07);--onacc:#06140c}",
      ".bm-w.bm-light{--bg:#fff;--panel:#f5f5f7;--text:#1d1d1f;--muted:#86868b;--bd:rgba(0,0,0,.08);--hover:rgba(0,0,0,.035);--onacc:#06140c}",
      ".bm-w *{box-sizing:border-box}",
      ".bm-w h3{margin:0 0 5px;font-size:20px;font-weight:600;letter-spacing:-.02em}",
      ".bm-w .bm-sub{margin:0 0 16px;font-size:13.5px;line-height:1.45;color:var(--muted);letter-spacing:-.01em}",
      ".bm-w label{display:block;font-size:12px;font-weight:600;margin:13px 0 6px;letter-spacing:-.01em}",
      ".bm-w input,.bm-w textarea{width:100%;border:1px solid transparent;border-radius:13px;padding:13px 15px;font-size:15px;font-family:inherit;color:var(--text);background:var(--panel);outline:none;transition:border-color .2s,box-shadow .2s}",
      ".bm-w input::placeholder,.bm-w textarea::placeholder{color:var(--muted)}",
      ".bm-w input:focus,.bm-w textarea:focus{border-color:var(--acc);box-shadow:0 0 0 4px color-mix(in srgb,var(--acc) 22%,transparent)}",
      ".bm-form input:focus,.bm-form textarea:focus{border-color:var(--acc-form);box-shadow:0 0 0 4px color-mix(in srgb,var(--acc-form) 22%,transparent)}",
      ".bm-btn{width:100%;margin-top:18px;border:0;border-radius:14px;padding:14px;font-size:15px;font-weight:700;letter-spacing:-.01em;color:var(--onacc);cursor:pointer;transition:transform .12s ease,filter .2s}",
      ".bm-btn:hover{filter:brightness(1.07)}.bm-btn:active{transform:scale(.975)}",
      ".bm-btn:disabled{opacity:.5;cursor:default;transform:none}",
      ".bm-btn-form{background:var(--acc-form)}.bm-btn-cal{background:var(--acc)}",
      ".bm-sec+.bm-sec{margin-top:24px;border-top:.5px solid var(--bd);padding-top:20px}",
      ".bm-err{font-size:12.5px;color:#ff453a;margin-top:10px}",
      ".bm-empty,.bm-loading{font-size:13px;color:var(--muted);text-align:center;padding:26px 0}",
      ".bm-ok{padding:30px 8px;text-align:center;font-size:15px;line-height:1.5;font-weight:500;animation:bm-in .35s ease both}",
      ".bm-ok-check{width:60px;height:60px;border-radius:50%;background:var(--acc);color:var(--onacc);font-size:30px;line-height:60px;margin:0 auto 14px;box-shadow:0 8px 22px color-mix(in srgb,var(--acc) 40%,transparent)}",
      ".bm-ok-when{margin-top:10px;font-size:14px;color:var(--muted)}",
      ".bm-cal{min-height:452px;display:flex;flex-direction:column}",
      ".bm-cal-body{flex:1;animation:bm-in .26s ease both}",
      ".bm-cal-titlerow{display:flex;align-items:center;gap:10px;margin-bottom:14px}",
      ".bm-cal-title{font-size:20px;font-weight:600;letter-spacing:-.02em}",
      ".bm-back{border:0;background:var(--panel);color:var(--text);font-size:13px;font-weight:600;cursor:pointer;padding:6px 12px;border-radius:999px}",
      ".bm-steps{display:flex;gap:6px;margin-bottom:18px}",
      ".bm-step{flex:1;height:4px;border-radius:999px;background:var(--hover);transition:background .3s}",
      ".bm-step.bm-on{background:var(--acc)}.bm-step.bm-done{background:color-mix(in srgb,var(--acc) 55%,var(--hover))}",
      ".bm-row{display:flex;align-items:center;gap:13px;width:100%;text-align:left;border:.5px solid var(--bd);border-radius:15px;padding:14px 15px;margin-bottom:9px;background:var(--panel);color:var(--text);cursor:pointer;transition:transform .14s ease,border-color .2s,background .2s}",
      ".bm-row:hover{transform:translateY(-1px);border-color:var(--acc)}",
      ".bm-row-name{font-size:15px;font-weight:600;letter-spacing:-.01em}",
      ".bm-row-meta{font-size:12.5px;color:var(--muted);margin-top:1px}",
      ".bm-col{display:flex;flex-direction:column;flex:1;min-width:0}",
      ".bm-pill{font-size:12px;font-weight:600;color:var(--muted);background:var(--hover);border-radius:999px;padding:5px 11px}",
      ".bm-chev{color:var(--muted);font-size:22px;line-height:1}",
      ".bm-av{display:inline-flex;align-items:center;justify-content:center;width:40px;height:40px;flex:0 0 auto;border-radius:50%;background:color-mix(in srgb,var(--acc) 22%,transparent);color:var(--acc);font-size:14px;font-weight:700}",
      ".bm-rec{display:flex;align-items:center;gap:12px;width:100%;text-align:left;border:1px solid var(--acc);border-radius:16px;padding:14px 15px;margin-bottom:12px;cursor:pointer;color:var(--text);background:color-mix(in srgb,var(--acc) 12%,var(--bg));transition:transform .14s ease,filter .2s}",
      ".bm-rec:hover{transform:translateY(-1px);filter:brightness(1.04)}",
      ".bm-rec-spark{font-size:20px}",
      ".bm-rec-top{font-size:10.5px;font-weight:700;text-transform:uppercase;letter-spacing:.05em;color:var(--acc)}",
      ".bm-rec-main{font-size:15px;font-weight:700;letter-spacing:-.01em}",
      ".bm-rec-sub{font-size:12.5px;color:var(--muted)}",
      ".bm-rec-go{margin-left:auto;font-size:12.5px;font-weight:700;color:var(--acc);white-space:nowrap}",
      ".bm-or{text-align:center;font-size:11.5px;color:var(--muted);margin:4px 0 12px;position:relative}",
      ".bm-month-nav{display:flex;align-items:center;justify-content:space-between;margin-bottom:10px}",
      ".bm-month-lbl{font-size:15px;font-weight:600;text-transform:capitalize}",
      ".bm-nav{width:34px;height:34px;border-radius:50%;border:0;background:var(--panel);color:var(--text);font-size:17px;cursor:pointer;transition:background .2s}",
      ".bm-nav:hover{background:var(--hover)}.bm-nav:disabled{opacity:.3;cursor:default}",
      ".bm-grid{display:grid;grid-template-columns:repeat(7,1fr);gap:4px}",
      ".bm-grid-h{margin-bottom:4px}",
      ".bm-wd{text-align:center;font-size:11px;font-weight:600;color:var(--muted);padding:4px 0}",
      ".bm-day{position:relative;aspect-ratio:1;border:0;background:transparent;color:var(--text);font-size:14px;font-weight:500;border-radius:50%;cursor:pointer;transition:background .15s}",
      ".bm-day span{position:relative;z-index:1}",
      ".bm-day:hover:not(.bm-off):not(.bm-sel){background:var(--hover)}",
      ".bm-day.bm-off{color:color-mix(in srgb,var(--text) 28%,transparent);cursor:default}",
      ".bm-day.bm-today{color:var(--acc);font-weight:700}",
      ".bm-day.bm-sel{background:var(--acc);color:var(--onacc);font-weight:700}",
      ".bm-day.bm-sel.bm-today{color:var(--onacc)}",
      ".bm-empty-day{cursor:default}",
      ".bm-dot{position:absolute;left:50%;bottom:5px;transform:translateX(-50%);width:4px;height:4px;border-radius:50%;background:var(--acc)}",
      ".bm-day.bm-sel .bm-dot{background:var(--onacc)}",
      ".bm-slots{display:grid;grid-template-columns:repeat(4,1fr);gap:8px}",
      ".bm-slot{border:.5px solid var(--bd);border-radius:12px;padding:11px 0;font-size:14px;font-weight:600;text-align:center;background:var(--panel);color:var(--text);cursor:pointer;transition:transform .12s ease,background .2s,color .2s,border-color .2s}",
      ".bm-slot:hover{transform:translateY(-1px);border-color:var(--acc);background:var(--acc);color:var(--onacc)}",
      ".bm-summary{background:color-mix(in srgb,var(--acc) 10%,var(--bg));border:.5px solid color-mix(in srgb,var(--acc) 30%,transparent);border-radius:15px;padding:14px 15px;font-size:14px;line-height:1.55;letter-spacing:-.01em;margin-bottom:6px}",
    ].join("");
  }
})();
