#!/usr/bin/env python3
"""
Скрипт генерации веб-ассетов (иконки, логотипы, баннеры) для проекта Next-Gen OPDS Suite («Боян»).
Берет исходники из папки media/ и генерирует набор оптимизированных файлов для фронтендов и документации.
"""

import os
from pathlib import Path
from PIL import Image, ImageDraw, ImageOps

ROOT = Path(__file__).resolve().parent.parent
MEDIA_DIR = ROOT / "media"
DESKTOP_PUB = ROOT / "frontends" / "web-desktop" / "public"
MOBILE_PUB = ROOT / "frontends" / "web-mobile" / "public"
DOCS_ASSETS = ROOT / "docs" / "assets"

def ensure_dirs():
    for d in [DESKTOP_PUB, MOBILE_PUB, DOCS_ASSETS]:
        d.mkdir(parents=True, exist_ok=True)

def create_circular_logo(src_img: Image.Image) -> Image.Image:
    """Создает круглый медальон с прозрачным альфа-каналом высокого качества."""
    w, h = src_img.size
    # Преобразуем в RGBA
    rgba = src_img.convert("RGBA")
    
    # Создаем маску высокого разрешения с небольшим безопасным отступом по контуру медальона
    mask = Image.new("L", (w, h), 0)
    draw = ImageDraw.Draw(mask)
    draw.ellipse((22, 22, w - 22, h - 22), fill=255)
    
    # Накладываем маску
    output = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    output.paste(rgba, (0, 0), mask=mask)
    return output

def create_maskable_icon(round_logo: Image.Image, size: int = 512) -> Image.Image:
    """
    Создает Android maskable иконку с отступом Safe Zone (~15%).
    Фон - благородный тёмный оттенок дерева/папируса (#1c140d).
    """
    canvas = Image.new("RGBA", (size, size), (28, 20, 13, 255))
    safe_size = int(size * 0.82)
    offset = (size - safe_size) // 2
    resized_logo = round_logo.resize((safe_size, safe_size), Image.Resampling.LANCZOS)
    canvas.paste(resized_logo, (offset, offset), mask=resized_logo)
    return canvas

def generate_all():
    ensure_dirs()
    
    sq_path = MEDIA_DIR / "Boyan_1-1.jpg"
    full_path = MEDIA_DIR / "Boyan_full.jpg"
    
    if not sq_path.exists() or not full_path.exists():
        print(f"Ошибка: Не найдены исходные файлы в {MEDIA_DIR}")
        return

    print("1. Обработка квадратного логотипа (Boyan_1-1.jpg)...")
    src_sq = Image.open(sq_path)
    round_logo = create_circular_logo(src_sq)
    
    # Базовые мастер-разрешения
    logo_512 = round_logo.resize((512, 512), Image.Resampling.LANCZOS)
    logo_256 = round_logo.resize((256, 256), Image.Resampling.LANCZOS)
    logo_192 = round_logo.resize((192, 192), Image.Resampling.LANCZOS)
    logo_180 = round_logo.resize((180, 180), Image.Resampling.LANCZOS) # Apple Touch Icon
    logo_64 = round_logo.resize((64, 64), Image.Resampling.LANCZOS)
    logo_48 = round_logo.resize((48, 48), Image.Resampling.LANCZOS)
    logo_32 = round_logo.resize((32, 32), Image.Resampling.LANCZOS)
    logo_16 = round_logo.resize((16, 16), Image.Resampling.LANCZOS)

    maskable_512 = create_maskable_icon(round_logo, 512)

    # Генерация favicon.ico (16, 32, 48)
    ico_targets = [DESKTOP_PUB / "favicon.ico", MOBILE_PUB / "favicon.ico"]
    for target in ico_targets:
        logo_48.save(
            target,
            format="ICO",
            sizes=[(16, 16), (32, 32), (48, 48)],
            append_images=[logo_32, logo_16]
        )
    print("  -> favicon.ico создан для desktop и mobile")

    # Генерация PNG-иконок для обоих фронтендов
    for pub_dir, name in [(DESKTOP_PUB, "web-desktop"), (MOBILE_PUB, "web-mobile")]:
        logo_256.save(pub_dir / "logo.png", "PNG", optimize=True)
        logo_180.save(pub_dir / "apple-touch-icon.png", "PNG", optimize=True)
        logo_32.save(pub_dir / "favicon-32x32.png", "PNG", optimize=True)
        logo_16.save(pub_dir / "favicon-16x16.png", "PNG", optimize=True)
        print(f"  -> Иконки сохранены в {name}/public")

    # Специфичные PWA иконки для mobile
    logo_192.save(MOBILE_PUB / "pwa-192x192.png", "PNG", optimize=True)
    logo_512.save(MOBILE_PUB / "pwa-512x512.png", "PNG", optimize=True)
    maskable_512.save(MOBILE_PUB / "pwa-maskable-512x512.png", "PNG", optimize=True)
    print("  -> PWA-иконки (192, 512, maskable) сохранены в web-mobile/public")

    # Сохранение логотипа в docs/assets
    logo_512.save(DOCS_ASSETS / "logo.png", "PNG", optimize=True)

    # Сохранение копий мастер-ассетов прямо в media/ для удобного доступа
    round_logo.save(MEDIA_DIR / "Boyan_logo_round_2048.png", "PNG", optimize=True)
    logo_512.save(MEDIA_DIR / "Boyan_logo_512.png", "PNG", optimize=True)
    logo_256.save(MEDIA_DIR / "Boyan_logo_256.png", "PNG", optimize=True)
    maskable_512.save(MEDIA_DIR / "Boyan_pwa_maskable_512.png", "PNG", optimize=True)
    logo_48.save(
        MEDIA_DIR / "favicon.ico",
        format="ICO",
        sizes=[(16, 16), (32, 32), (48, 48)],
        append_images=[logo_32, logo_16]
    )
    print("  -> Мастер-ассеты сохранены в media/ (Boyan_logo_round_2048.png, Boyan_logo_512.png, favicon.ico и др.)")

    print("\n2. Обработка широкого баннера (Boyan_full.jpg)...")
    src_banner = Image.open(full_path).convert("RGB")
    
    # Масштабируем до 1920 по ширине с сохранением пропорций
    orig_w, orig_h = src_banner.size
    target_w = 1920
    target_h = int(orig_h * (target_w / orig_w))
    banner_resized = src_banner.resize((target_w, target_h), Image.Resampling.LANCZOS)

    banner_out = DOCS_ASSETS / "banner.jpg"
    banner_resized.save(banner_out, "JPEG", quality=85, optimize=True, progressive=True)
    banner_size_kb = os.path.getsize(banner_out) / 1024
    print(f"  -> docs/assets/banner.jpg сохранен ({target_w}x{target_h}, {banner_size_kb:.1f} KB)")

    # Сохранение оптимизированного баннера также в media/
    banner_resized.save(MEDIA_DIR / "Boyan_banner_1920.jpg", "JPEG", quality=85, optimize=True, progressive=True)
    print("  -> media/Boyan_banner_1920.jpg сохранен")

    print("\nГенерация всех веб-ассетов успешно завершена!")

if __name__ == "__main__":
    generate_all()
