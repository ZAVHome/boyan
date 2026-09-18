# User Guide: Boyan

<img src="assets/logo.svg" align="right" width="90" alt="Boyan" />

Welcome to the comprehensive user guide for **Boyan** — an ultra-lightweight, modular eBook catalog, in-browser reading suite, and OPDS distribution server.

---

## 📑 Table of Contents

1. [Introduction & Key Features](#1-introduction--key-features)
2. [Desktop & Tablet Web Interface (Web Desktop)](#2-desktop--tablet-web-interface-web-desktop)
   - [Catalog Browsing & Search](#catalog-browsing--search)
   - [Built-in Universal Web Reader](#built-in-universal-web-reader)
   - [Virtual Shelves & Reading Progress](#virtual-shelves--reading-progress)
   - [Duplicate & Quarantine Moderation](#duplicate--quarantine-moderation)
3. [Mobile PWA Application (Web Mobile)](#3-mobile-pwa-application-web-mobile)
   - [Installing the PWA on Android and iOS](#installing-the-pwa-on-android-and-ios)
   - [Touch Gestures & Navigation](#touch-gestures--navigation)
   - [Offline Library (Zero-Connectivity Reading)](#offline-library-zero-connectivity-reading)
   - [E-Ink High-Contrast Mode for E-Paper Readers](#e-ink-high-contrast-mode-for-e-paper-readers)
4. [Connecting External E-Readers via OPDS](#4-connecting-external-e-readers-via-opds)
   - [Feed URLs](#feed-urls)
   - [PocketBook Setup](#pocketbook-setup)
   - [AlReaderX Setup](#alreaderx-setup)
   - [KOReader Setup](#koreader-setup)
   - [Moon+ Reader Setup (Android)](#moon-reader-setup-android)
5. [Using the Telegram Bot](#5-using-the-telegram-bot)
   - [Searching and Downloading Books](#searching-and-downloading-books)
   - [Uploading Books to Your Library via Chat](#uploading-books-to-your-library-via-chat)
6. [Frequently Asked Questions (FAQ)](#6-frequently-asked-questions-faq)

---

## 1. Introduction & Key Features

Boyan is engineered to provide a seamless digital reading experience across all your devices, combining home library cataloging with ultra-fast network distribution:

- **Supported Formats:** `.fb2`, `.fb2.zip`, `.epub`, `.mobi`, `.pdf`, `.djvu`.
- **Dual Modern Frontends:**
  - **Desktop SPA:** High-density catalog view, detailed book modals, interactive web reader, and administrative tools.
  - **Mobile PWA:** Gesture-first, one-handed navigation, offline caching, and specialized E-Paper display profiles.
- **Dual OPDS Catalogs:**
  - **OPDS v1.2 (XML/Atom):** 100% compatible with hardware e-readers (PocketBook, Onyx Boox) and popular mobile apps.
  - **OPDS v2.0 (JSON-LD):** Built for next-generation modern reading clients.
- **Two-Way Telegram Bot:** Instant search and download directly in Telegram, plus one-tap book ingestion by dropping files into the bot chat.
- **Visual Themes:** Light, Dark, OLED Pure Black, Warm Sepia, and Zero-Animation E-Ink contrast mode.

---

## 2. Desktop & Tablet Web Interface (Web Desktop)

The desktop application is accessible in any modern web browser at: **`http://localhost:3000`** (or your server's host IP/domain).

### Catalog Browsing & Search

- **Global Search:** Type book titles, author names, or series titles in the top navigation bar. Search features SQLite FTS5 with built-in stemming for English and Russian.
- **Storefront Sorting:** Toggle display order — *Recent Additions*, *By Title (A-Z)*, *By Author (A-Z)*.
- **Interactive Filters & Badges:** Clicking any author, series, genre tag, publisher, publication year, or language instantly filters the catalog and adds active filter badges with one-click removal `×`.
- **Book Cards:** Display high-resolution cover thumbnails, book title, clickable author names, series with index (`#1`), and available file format badges.
- **Book Detail Modal:** Click any book card to inspect complete annotations, tags, publication metadata (publisher, year, language, ISBN, file size), and action buttons:
  - **Read Online** — launches the universal reader directly in your browser.
  - **Download** — downloads the original book archive (`.fb2`, `.fb2.zip`, or `.epub`).
  - **Add to Shelf** — organizes the book into your personal reading shelves.
  - **Edit Metadata (Admin)** — quick navigation to the administrative book metadata editor.

### Authors & Series Catalogs

- **Authors Catalog (`/authors`):**
  - Alphabetical index (A-Z, А-Я) for fast jumping to specific letters.
  - Live search filtering authors by name substring.
  - Author cards showing total book counts.
  - Dedicated author detail page displaying their bibliography with one-click reading.
- **Series & Cycles Catalog (`/series`):**
  - Alphabetical index and search across all book series.
  - Series cards showing total volume count.
  - Displays all volumes in proper chronological volume order (`#1`, `#2`, `#3`...).

### Connect Reader Modal (ConnectModal)

The top navigation bar and sidebar provide a **"Connect Reader"** button:
- Displays instant OPDS v1.2 (Atom XML), OPDS v2.0 (JSON-LD), and REST API documentation URLs.
- Includes **Copy** buttons for one-click clipboard copying to paste into e-reader apps.

### Built-in Universal Web Reader

The in-browser reader automatically detects the format and delivers optimal typography:

- **For FB2 and FB2.ZIP:** Instant chapter streaming with proper formatting, cover page display, and built-in XML markup sanitization.
- **For EPUB:** Smooth pagination powered by ePub.js with dynamic theme styling.

#### Reader Settings (Gear Icon in Top Toolbar)

| Setting | Options | Description |
| :--- | :--- | :--- |
| **Theme** | Light, Sepia, Dark, OLED | Adjusts color palette to ambient lighting conditions |
| **Font Size** | 12px to 32px | Fine-tune text size with `A-` / `A+` controls |
| **Font Family** | Serif, Sans-Serif, Monospace | Switch between classic book typography and modern sans-serif fonts |
| **Line Spacing** | 1.2, 1.5, 1.8, 2.0 | Customize line height to minimize eye strain |
| **Margins** | Narrow, Normal, Wide | Adapts reading column width to widescreen monitors |

> [!TIP]
> Your reading progress (percentage and chapter) is continuously synchronized with the server and restored when you open the book on any device.

### Virtual Shelves & Reading Progress

Organize your library with built-in user shelves:

1. **Reading (`reading`)** — Books you are currently reading.
2. **Want to Read (`to-read`)** — Your reading backlog and wishlist.
3. **Finished (`finished`)** — Completed books.

To change shelf status, open any book modal or use the quick shelf dropdown selector.

### Admin Suite (`/admin`)

Users with the `admin` role can access the comprehensive administrative console at **`/admin`**:

1. **Dashboard & Host Metrics:** Live server resource monitoring (RAM usage, CPU, goroutines, uptime), SQLite WAL database status, cover cache size, and Telegram bot connectivity.
2. **User Management:** Create accounts, assign roles (`admin`, `user`, `restricted`), reset passwords, ban, or delete users.
3. **Book Curation & Metadata Editor (`/admin/books/:id/edit`):**
   - Inspect technical parameters (storage path, file size, SHA-256 hash).
   - Edit title, original title, annotation, publisher, year, language, and ISBN.
   - Manage authors, assign series with sequence numbers, edit genre tags.
   - Preview book cover with a one-click button to regenerate the cover thumbnail from source files.
   - Batch deletion and curation from the books list table.
4. **Storage & Task Manager:**
   - Storage utilization metrics.
   - Background folder scanning (`watch_dir` and `library_dir`).
   - Background FB2 repair and sanitization (repairs BOM, missing XML namespaces, and unescaped HTML entities across existing files).
   - Asynchronous Calibre library import with real-time progress indicators.
   - SQLite WAL truncation (`PRAGMA wal_checkpoint(TRUNCATE)`) and cover cache purging.
5. **System Logs:** Live viewer of recent `slog` server events with level filtering (INFO, WARN, ERROR, DEBUG).
6. **System Configuration:** In-browser editor for `config.yaml` with hot-reload of services.
7. **Quarantine Moderation:** Review quarantined duplicate or corrupt files with force-restore or permanent deletion.

---

## 3. Mobile PWA Application (Web Mobile)

The mobile client is accessible at: **`http://localhost:3001`**. It is optimized for one-handed smartphone use and Android-based E-Ink devices.

### Installing the PWA on Android and iOS

Boyan is a Progressive Web App (PWA) that installs as a native application without browser URL bars:

- **Android (Google Chrome / Samsung Internet):**
  1. Open `http://<YOUR-SERVER-IP>:3001` in your browser.
  2. Tap the bottom banner **"Install Boyan App"** or tap browser menu `⋮` → **"Add to Home screen"** / **"Install App"**.
  3. The Boyan app icon will appear on your app drawer and home screen.
- **iOS (Apple Safari):**
  1. Open `http://<YOUR-SERVER-IP>:3001` in Safari.
  2. Tap the **Share** button (the square icon with an upward arrow at the bottom).
  3. Scroll down and tap **"Add to Home Screen"** → tap **"Add"**.

### Touch Gestures & Navigation

The mobile reader is designed for touch interaction:

- **Turn Pages:** Swipe left for next page, swipe right for previous page.
- **Show Controls:** Tap the **center third of the screen** to toggle top and bottom toolbars (table of contents, progress slider, and typography settings).
- **Book Action Sheet:** Tap the `⋮` button on any book card to bring up a bottom action sheet with fast actions (*Read*, *Save Offline*, *Change Shelf*).

### Offline Library (Zero-Connectivity Reading)

Take your favorite books anywhere, even without Wi-Fi or cellular service:

1. Tap the **Download Offline** button (downward arrow icon) on any book card.
2. The entire book payload and its cover are cached inside your device's browser database (`IndexedDB`).
3. Navigate to the **Offline** tab in the bottom navigation bar:
   - All stored books are listed and readable with zero network connectivity.
   - Tap **"Remove"** at any time to reclaim local storage space.

### E-Ink High-Contrast Mode for E-Paper Readers

For specialized Android e-readers (Onyx Boox, PocketBook, Likebook, Meebook):

1. Open the reader menu or side drawer and choose the **"E-Ink"** theme.
2. **Key E-Ink Optimizations:**
   - Background is forced to pure white (`#ffffff`) and text to pure black (`#000000`).
   - All CSS transitions, box-shadows, blurs, and animations are strictly disabled.
   - Eliminates ghosting artifacts and significantly extends e-reader battery life.

---

## 4. Connecting External E-Readers via OPDS

Boyan exposes standard OPDS feeds compatible with all major hardware and software eBook readers.

### Feed URLs

Replace `<SERVER_IP>` with your host machine's local network IP address (e.g., `192.168.1.50`):

| Protocol | Feed URL | Purpose |
| :--- | :--- | :--- |
| **OPDS v1.2 (XML/Atom)** | `http://<SERVER_IP>:8080/opds/v1/feed.xml` | Standard feed for PocketBook, AlReaderX, KOReader, Moon+ Reader |
| **OPDS v2.0 (JSON-LD)** | `http://<SERVER_IP>:8080/opds/v2/catalog.json` | Next-gen feed for modern clients |
| **OpenSearch Descriptor** | `http://<SERVER_IP>:8080/opds/v1/search.xml` | Automatically linked for in-app catalog searches |

---

### PocketBook Setup

1. Connect your PocketBook to the same Wi-Fi network as the Boyan server.
2. Open the **Network Libraries** (or **OPDS Catalogs**) app.
3. Tap **Add Catalog**.
4. In the **Name** field, enter: `Boyan`.
5. In the **URL** field, enter:

   ```text
   http://192.168.1.50:8080/opds/v1/feed.xml
   ```

6. Tap **Save** and open the catalog to browse and download books directly to your device storage.

---

### AlReaderX Setup

1. Open AlReaderX and go to **Network Catalogs**.
2. Tap **Add Catalog**.
3. Set the name to `Boyan`.
4. Enter the URL:

   ```text
   http://192.168.1.50:8080/opds/v1/feed.xml
   ```

5. Confirm with **OK** to browse by authors, genres, or series.

---

### KOReader Setup

1. Open KOReader and select the magnifying glass icon or tap **OPDS Catalog**.
2. Tap **Add New Catalog**.
3. Fill in:
   - **Title:** `Boyan`
   - **URL:** `http://192.168.1.50:8080/opds/v1/feed.xml`
4. Tap **OK** to load your library tree.

---

### Moon+ Reader Setup (Android)

1. Open Moon+ Reader and switch to the **Net Library** tab.
2. Tap the three dots menu icon (`⋮`) in the top right → **Add New Catalog**.
3. Provide the name (`Boyan`) and URL:

   ```text
   http://192.168.1.50:8080/opds/v1/feed.xml
   ```

4. Tap **OK**.

---

## 5. Using the Telegram Bot

The built-in Telegram bot offers bidirectional communication: search and download books, or upload new files directly from your smartphone.

### Searching and Downloading Books

1. Open your Telegram chat with the bot and send `/start`.
2. Type any search term (e.g. `Asimov` or `Foundation`) or use the command:

   ```text
   /search Foundation
   ```

3. The bot responds with matching titles, authors, and series descriptions.
4. Inline buttons below each result display available download options (e.g., `📥 Download FB2.ZIP`, `📥 Download EPUB`).
5. Tap the button for your preferred format; the bot will send the eBook file directly into your chat.

### Uploading Books to Your Library via Chat

To add a book to your collection on the go:

1. Send or forward any eBook document (`.fb2`, `.zip`, `.epub`, `.mobi`, `.pdf`) directly into the bot chat.
2. The bot downloads the file, processes metadata, and adds it to the catalog:
   - **New Book:** The bot parses authors, genres, and cover images, responding: *"Book successfully added to the catalog!"*.
   - **Duplicate Book:** The bot detects identical SHA-256 hashes and warns: *"This book already exists in your library (duplicate moved to quarantine)"*.

---

## 6. Frequently Asked Questions (FAQ)

### Why does downloading a book inside a `.fb2.zip` archive feel instantaneous?

Boyan parses and streams nested archives directly from RAM using Go buffers. It never writes temporary extracted files to disk.

### Can I sync reading progress between my phone and computer?

Yes. When logged into your account, reading position and shelf statuses are saved in the central SQLite database and synced across all clients.

### What should I do if an uploaded book does not appear in search?

1. Ensure the file extension is supported (`.fb2`, `.fb2.zip`, `.epub`).
2. Check the **Quarantine** section in the web interface: if the XML was invalid or the file was an identical duplicate, it was moved to quarantine.
3. Verify that the book storage directory is properly mounted and configured in `config.yaml`.
