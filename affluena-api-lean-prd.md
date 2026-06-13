# Product Requirements Document (PRD)

**Project Name:** Affluena-API
**Phase:** Minimum Viable Product (MVP)
**Tech Stack:** Go (Fiber/Gin), PostgreSQL, Docker.
**Auth:** Native JWT Authentication (Golang).
**Background Tasks:** Native Go Scheduler (e.g., `gocron` / `robfig/cron`).

## 1. Product Overview

**Affluena-API** adalah sistem pencatatan keuangan API-first yang dirancang untuk kecepatan pencatatan harian dan manajemen portofolio aset yang komprehensif. Aplikasi ini memiliki kapabilitas _native_ untuk melacak cicilan berjangka, manajemen _subscription_, pemisahan dompet fiat dan instrumen _trading_, serta memfasilitasi transaksi otomatis berulang tanpa bergantung pada _message broker_ eksternal. Sistem ini juga dilengkapi dengan mekanisme observabilitas penuh (API Logging) untuk perekaman setiap aktivitas jaringan yang masuk.

## 2. Objectives

- Membangun fondasi _backend_ (API) yang _scalable_, aman, dan konsisten (ACID compliant).
- Menyistemasi autentikasi dan manajemen otorisasi sepenuhnya di level aplikasi (Go) tanpa dependensi _third-party_ seperti Keycloak.
- Menyederhanakan infrastruktur dengan menggunakan _native scheduler_ di Go untuk mengeksekusi _background jobs_ (menggantikan RabbitMQ).
- Menyediakan _endpoint_ API yang siap dikonsumsi oleh Progressive Web App (PWA) di masa depan.
- Menciptakan transparansi dan _auditability_ sistem melalui pencatatan log lengkap (hingga payload request/response).

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
- **Category Hierarchy:** Kategori dapat memiliki `parent_id` hingga 3 level. Parent kategori harus dimiliki user yang sama dan memiliki tipe yang sama (`income` dengan `income`, `expense` dengan `expense`), serta tidak boleh membentuk siklus.
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

### 3.8. Financial Goals (Tabungan Bersama) & Shared Wallets

- **Fitur:** Menetapkan target tabungan finansial dengan batas waktu, dan kemampuan mengundang pengguna lain sebagai anggota (kolaborasi). Termasuk fungsionalitas berbagi "Shared Wallet" untuk pengeluaran operasional gabungan.
- **Use Case:** Suami dan istri bisa memiliki satu dompet gabungan khusus pengeluaran belanja dapur. Keduanya bebas mencatat transaksi menggunakan dompet yang di-*share* sehingga *cashflow* dan *dashboard analytics* menjadi lebih akurat di kedua belah pihak.
- **Integrity Rules:** Pembuatan goal dan penerimaan undangan membuat _Goal Wallet_ secara atomik. Nama goal boleh sama karena _Goal Wallet_ memakai suffix ID goal. Undangan yang ditolak tidak lagi membuka akses goal, dan respons undangan hanya boleh dilakukan oleh member yang sesuai dengan `:user_id` pada route.

### 3.9. Tags / Labeling

- **Fitur:** Menyematkan label (_hashtag_) lintas-kategori secara tak terbatas pada transaksi, dengan arsitektur _Many-to-Many_.
- **Use Case:** Melacak total pengeluaran untuk suatu acara spesifik (misal liburan `#Bali2026` atau uang klaim `#ReimburseKantor`) tanpa harus merusak struktur kategori utama (`Makanan`, `Transportasi`, dsb). Pengguna dapat memfilter mutasi uang berdasarkan `tag_id`.
- **Integrity Rules:** `tag_id` dan `tag_ids` harus valid, dimiliki oleh pengguna yang sedang login, dan tidak boleh menautkan transaksi ke label milik user lain. Duplikasi `tag_ids` dalam satu request disimpan satu kali.

### 3.10. API Observability & Audit Trail

- **Fitur:** Mencatat riwayat akses ke server API secara transparan ke dalam _database_, mencakup IP, User Agent, durasi latency, payload request, hingga payload response.
- **Use Case:** Developer menggunakan histori ini untuk memantau keamanan (serangan ke sistem), melakukan proses _debugging_ jika terjadi masalah _request_ (_payload_ tidak masuk akal), dan memastikan SLA (_Service Level Agreement_) API terjaga dari anomali _latency_. Keamanan password dan token pada *endpoint authentication* tetap disensor (masked), termasuk payload response yang berisi access/refresh token.

## 4. Non-Functional Requirements

- **Performance:** _Response time_ untuk _endpoint_ krusial (_Quick Entry_) di bawah 200ms.
- **Data Integrity:** Transaksi yang memengaruhi banyak tabel (misalnya bayar hutang yang mengubah _balance wallet_ dan _status_ hutang) wajib dibungkus dalam PostgreSQL _Database Transaction_ (Commit/Rollback).
- **Simplicity:** Arsitektur yang mandiri (tanpa Keycloak/RabbitMQ) membuat proses _deployment_ via Docker menjadi jauh lebih ramping dan hemat _resource_ memori, sangat optimal dijalankan pada _instance server_ berukuran kecil.

## 5. Out of Scope (For MVP)

- Antarmuka pengguna visual (Frontend/UI/PWA).
- Integrasi _payment gateway_ pihak ketiga atau akses mutasi API Bank _direct_.
- Fitur OCR (_Optical Character Recognition_) untuk _scan_ struk otomatis.

### 3.11. Split Bill (Macro Transaction)

- **Fitur:** Memfasilitasi pembayaran tagihan gabungan dalam satu transaksi, lalu memecahnya secara otomatis menjadi pengeluaran pribadi dan piutang ke teman.
- **Use Case:** Pengguna membayar tagihan makan malam Rp 300.000 dengan teman-temannya. Ia membuat satu request Split Bill, dan API secara cerdas akan mencatat Rp 100.000 sebagai `Expense` miliknya, serta Rp 200.000 sebagai `Disbursement` untuk membuat 2 entitas `Debt` (Piutang) baru bagi temannya.

### 3.12. Mailer & Notifications

- **Fitur:** Mengirimkan peringatan melalui email asinkron berbasis SMTP jika terdapat parameter keuangan yang mengkhawatirkan.
- **Use Case:** Saat pengguna mencatatkan transaksi pengeluaran (misal belanja makanan), di _background_, _scheduler/goroutine_ internal mendeteksi bahwa pengeluaran kategori Makanan sudah mencapai >80% atau >100% dari limit bulanan, kemudian mengirim peringatan HTML elegan via Mailtrap / SendGrid agar pengguna lebih berhati-hati.

### 3.13 Advanced Analytics & Reporting
Sistem menyediakan analitik mendalam untuk memantau tren dan status keuangan.

- **Cashflow Trend**: Mendapatkan data income dan expense selama 1-12 bulan ke belakang untuk melihat pergerakan cashflow.
- **Expense Distribution**: Melihat distribusi pengeluaran berdasarkan kategori dalam bentuk persentase.
- **Spend Forecasting**: Sistem menghitung rata-rata harian pengeluaran dan memprediksi total pengeluaran di akhir bulan, serta memberikan peringatan ("overbudget") jika prediksi melewati batas budget yang dianggarkan. Jika belum ada budget pada bulan tersebut, status tetap "safe".
