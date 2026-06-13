# Product Requirements Document (PRD)

**Project Name:** Affluena-API
**Phase:** Minimum Viable Product (MVP)
**Tech Stack:** Go (Fiber/Gin), PostgreSQL, Docker.
**Auth:** Native JWT Authentication (Golang).
**Background Tasks:** Native Go Scheduler (e.g., `gocron` / `robfig/cron`).

## 1. Product Overview

**Affluena-API** adalah sistem pencatatan keuangan API-first yang dirancang untuk kecepatan pencatatan harian dan manajemen portofolio aset yang komprehensif. Aplikasi ini memiliki kapabilitas _native_ untuk melacak cicilan berjangka, manajemen _subscription_, pemisahan dompet fiat dan instrumen _trading_, serta memfasilitasi transaksi otomatis berulang tanpa bergantung pada _message broker_ eksternal.

## 2. Objectives

- Membangun fondasi _backend_ (API) yang _scalable_, aman, dan konsisten (ACID compliant).
- Menyistemasi autentikasi dan manajemen otorisasi sepenuhnya di level aplikasi (Go) tanpa dependensi _third-party_ seperti Keycloak.
- Menyederhanakan infrastruktur dengan menggunakan _native scheduler_ di Go untuk mengeksekusi _background jobs_ (menggantikan RabbitMQ).
- Menyediakan _endpoint_ API yang siap dikonsumsi oleh Progressive Web App (PWA) di masa depan.

## 3. Core Features & Use Cases (MVP)

### 3.1. Authentication & Security (Native)

Autentikasi dikelola secara independen di dalam _backend_ Affluena-API.

- **Fitur:** _Register_, _Login_, dan pelindungan _endpoint_ menggunakan JWT (JSON Web Tokens).
- **Use Case:** Pengguna melakukan _login_ dengan _email_ dan _password_. Server memvalidasi _hash_ (menggunakan _bcrypt_), lalu menerbitkan _Access Token_ dan _Refresh Token_ untuk menjaga sesi tetap aktif di PWA.

### 3.2. Wallet & Multi-Asset Management

Sistem mendukung lebih dari satu dompet dengan tipe aset yang berbeda untuk melacak _Net Worth_ secara utuh.

- **Tipe Wallet:** Cash, Bank, E-Wallet, dan Investment/Trading.
- **Use Case:** Pemisahan _wallet_ untuk rekening bank operasional harian dan _wallet_ khusus untuk memantau saldo _trading_ MT5 (XAUUSD) atau aset _crypto_. Mutasi saldo antar dompet (seperti _deposit/withdrawal_ ke _broker_) dicatat murni sebagai transfer, bukan _Expense_ atau _Income_.

### 3.3. Transaction & Quick Entry Templates

Manajemen mutasi masuk, keluar, dan penyesuaian saldo.

- **Fitur:** CRUD transaksi mencakup nominal, _wallet_id_, _category_id_, tanggal, dan catatan tambahan.
- **Quick Entry:** _Endpoint_ khusus untuk transaksi instan berdasarkan _template_.
- **Use Case:** Eksekusi _template_ pengeluaran rutin dalam satu klik, seperti biaya tol dan transportasi _commute_ rute Tambun - SCBD, atau pengeluaran makan siang standar.

### 3.4. Installment & Subscription Tracker

Pelacakan pengeluaran yang memiliki tenor berjangka atau siklus berulang.

- **Installment (Cicilan):** Mencatat total tagihan, jumlah bulan cicilan, cicilan per bulan, dan sisa tenor.
    - _Use Case:_ Memantau sisa cicilan 3 bulan untuk langganan LeetCode Premium atau tagihan _membership_ FTL Gym SCBD. Sistem otomatis mengurangi sisa bulan setiap kali _trigger_ pembayaran diaktifkan.
- **Subscription (Langganan):** Pelacakan layanan dengan siklus pembayaran tertentu (mingguan/bulanan).
    - _Use Case:_ _Tracker_ rutin untuk _renewal_ paket _meal plan_ mingguan Yellowfit Protein+ agar alokasi dananya selalu disiapkan sebelum jatuh tempo.

### 3.5. Recurring Transactions (Native Automated Logging)

- **Fitur:** Menggunakan ekosistem _native cron job_ di dalam Go (`gocron` atau modul serupa) yang berjalan di _background goroutine_ untuk mengeksekusi transaksi otomatis.
- **Use Case:** Setiap tanggal 1, _scheduler_ internal otomatis memotong saldo _wallet_ utama untuk tagihan tetap seperti internet atau sewa tempat tinggal, mencatat baris transaksi baru secara presisi tanpa intervensi manual.

### 3.6. Debt & Loan Manager (Hutang Piutang)

- **Fitur:** Mencatat entitas peminjam/pemberi pinjaman, batas waktu, dan status progres pembayaran (Belum Lunas, Dicicil, Lunas).
- **Use Case:** Mengurangi saldo dompet saat memberikan pinjaman ke teman (masuk sebagai "Aset Piutang"). Saat cicilan dari teman tersebut masuk dan mencapai total nominal pinjaman, status otomatis berubah menjadi "Lunas".

### 3.7. Category Budgeting

- **Fitur:** Limit maksimal pengeluaran bulanan berdasarkan ID Kategori.
- **Use Case:** Memasang limit _budget_ khusus untuk pos _gaming/entertainment_ (misalnya belanja _game_ di Steam seperti Resident Evil 4 Remake atau _item_ Dota 2) serta pos _personal care_ (belanja stok rutin Skintific, Lanbena, dan produk rambut seperti Regrou/Erha). API akan mengembalikan data limit yang terpakai untuk dirender menjadi visualisasi _progress bar_ di klien.

### 3.8. Financial Goals (Tabungan Bersama)

- **Fitur:** Menetapkan target tabungan finansial dengan batas waktu, dan kemampuan mengundang pengguna lain sebagai anggota (kolaborasi).
- **Use Case:** Membuat target tabungan "Menikah" dengan target nominal tertentu. Pengguna bisa mengundang pasangannya, dan keduanya bisa mengalokasikan (menyisihkan) uang dari dompet pribadi ke dalam _Goal Wallet_ yang berelasi dengan target tabungan ini. Progres tabungan dapat terpantau bersama.

## 4. Non-Functional Requirements

- **Performance:** _Response time_ untuk _endpoint_ krusial (_Quick Entry_) di bawah 200ms.
- **Data Integrity:** Transaksi yang memengaruhi banyak tabel (misalnya bayar hutang yang mengubah _balance wallet_ dan _status_ hutang) wajib dibungkus dalam PostgreSQL _Database Transaction_ (Commit/Rollback).
- **Simplicity:** Arsitektur yang mandiri (tanpa Keycloak/RabbitMQ) membuat proses _deployment_ via Docker menjadi jauh lebih ramping dan hemat _resource_ memori, sangat optimal dijalankan pada _instance server_ berukuran kecil.

## 5. Out of Scope (For MVP)

- Antarmuka pengguna visual (Frontend/UI/PWA).
- Integrasi _payment gateway_ pihak ketiga atau akses mutasi API Bank _direct_.
- Fitur OCR (_Optical Character Recognition_) untuk _scan_ struk otomatis.
