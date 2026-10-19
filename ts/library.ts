import './theme';

interface LibraryAsset {
    id: number;
    ext: string;
    mime_type: string;
    size_bytes: number;
    original_name: string;
    title: string;
    description: string;
    kind: string;
    extension_id: number;
    extension_name?: string;
    track_id: string;
    track_name?: string;
    url: string;
    thumbnail_url: string;
    created_at: string;
}

let _libraryAssets: LibraryAsset[] = [];

function escapeHtml(value: unknown): string {
    return String(value == null ? '' : value).replace(/[&<>"']/g, (m) => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;',
    }[m] as string));
}

function formatBytes(bytes: number): string {
    if (!bytes || bytes < 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    let value = bytes;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
        value /= 1024;
        unit++;
    }
    const rounded = unit === 0 ? value : Math.round(value * 10) / 10;
    return `${rounded} ${units[unit]}`;
}

function kindBadge(kind: string): string {
    if (kind === 'map') return '<span class="badge bg-primary">Map</span>';
    if (kind === 'extension') return '<span class="badge bg-danger">Extension</span>';
    return '<span class="badge bg-secondary">Other</span>';
}

function assetCard(asset: LibraryAsset): string {
    const preview = asset.thumbnail_url
        ? `<a href="${escapeHtml(asset.url)}" target="_blank" rel="noopener" class="d-block">
               <img src="${escapeHtml(asset.thumbnail_url)}" alt="${escapeHtml(asset.title)}" loading="lazy" class="card-img-top" style="height:170px;object-fit:cover">
           </a>`
        : `<a href="${escapeHtml(asset.url)}" target="_blank" rel="noopener" class="d-flex align-items-center justify-content-center bg-light text-secondary" style="height:170px">
               <i class="fa-solid ${asset.mime_type === 'application/pdf' ? 'fa-file-pdf text-danger' : 'fa-file'} fa-4x"></i>
           </a>`;

    const associations: string[] = [];
    if (asset.extension_name) associations.push(`<span class="badge bg-light text-dark border"><i class="fa-solid fa-box-open me-1"></i>${escapeHtml(asset.extension_name)}</span>`);
    if (asset.track_name) associations.push(`<span class="badge bg-light text-dark border"><i class="fa-solid fa-route me-1"></i>${escapeHtml(asset.track_name)}</span>`);

    const description = asset.description
        ? `<p class="card-text small text-muted mb-2">${escapeHtml(asset.description)}</p>`
        : '';

    return `<div class="col-12 col-sm-6 col-lg-4 col-xl-3" data-kind="${escapeHtml(asset.kind)}" data-extension="${asset.extension_id}" data-search="${escapeHtml((asset.title + ' ' + asset.original_name).toLowerCase())}">
        <div class="card h-100">
            ${preview}
            <div class="card-body d-flex flex-column">
                <h5 class="card-title h6 mb-1">${escapeHtml(asset.title)}</h5>
                <div class="mb-2">${kindBadge(asset.kind)} ${associations.join(' ')}</div>
                ${description}
                <div class="mt-auto small text-muted d-flex justify-content-between align-items-center pt-2 border-top">
                    <span>${escapeHtml(asset.original_name)}</span>
                    <span class="text-nowrap">${formatBytes(asset.size_bytes)}</span>
                </div>
            </div>
            <div class="card-footer bg-transparent d-flex gap-2">
                <a class="btn btn-sm btn-outline-primary flex-fill" href="${escapeHtml(asset.url)}" target="_blank" rel="noopener"><i class="fa-solid fa-up-right-from-square me-1"></i>Open</a>
                <a class="btn btn-sm btn-outline-secondary flex-fill" href="/api/assets/${asset.id}/download"><i class="fa-solid fa-download me-1"></i>Download</a>
            </div>
        </div>
    </div>`;
}

function renderExtensionFilter(): void {
    const select = document.getElementById('library-extension') as HTMLSelectElement | null;
    if (!select) return;
    const current = select.value;
    const seen = new Map<number, string>();
    _libraryAssets.forEach((asset) => {
        if (asset.extension_id > 0 && !seen.has(asset.extension_id)) {
            seen.set(asset.extension_id, asset.extension_name || `Extension ${asset.extension_id}`);
        }
    });
    select.innerHTML = '<option value="">All extensions</option>' +
        Array.from(seen.entries()).map(([id, name]) => `<option value="${id}">${escapeHtml(name)}</option>`).join('');
    select.value = current;
}

function applyFilters(): void {
    const grid = document.getElementById('library-grid');
    if (!grid) return;
    const kind = (document.getElementById('library-kind') as HTMLSelectElement | null)?.value || '';
    const extensionID = (document.getElementById('library-extension') as HTMLSelectElement | null)?.value || '';
    const search = (document.getElementById('library-search') as HTMLInputElement | null)?.value.trim().toLowerCase() || '';

    let visible = 0;
    grid.querySelectorAll<HTMLElement>('[data-kind]').forEach((card) => {
        const matchesKind = !kind || card.dataset.kind === kind;
        const matchesExtension = !extensionID || card.dataset.extension === extensionID;
        const matchesSearch = !search || (card.dataset.search || '').includes(search);
        const show = matchesKind && matchesExtension && matchesSearch;
        card.style.display = show ? '' : 'none';
        if (show) visible++;
    });

    const existing = document.getElementById('library-empty');
    if (visible === 0) {
        if (!existing) {
            const empty = document.createElement('div');
            empty.id = 'library-empty';
            empty.className = 'col-12 text-center text-muted py-5';
            empty.innerHTML = '<i class="fa-solid fa-folder-open fa-3x mb-3 opacity-50"></i><p>No files match these filters.</p>';
            grid.appendChild(empty);
        }
    } else if (existing) {
        existing.remove();
    }
}

async function loadLibrary(): Promise<void> {
    const grid = document.getElementById('library-grid');
    if (!grid) return;

    try {
        const res = await fetch('/api/assets?limit=500');
        if (!res.ok) throw new Error(res.statusText);
        const assets = await res.json();
        _libraryAssets = Array.isArray(assets) ? assets : [];

        if (!_libraryAssets.length) {
            grid.innerHTML = '<div class="col-12 text-center text-muted py-5"><i class="fa-solid fa-folder-open fa-3x mb-3 opacity-50"></i><p>No maps or extension files have been uploaded yet.</p></div>';
            renderExtensionFilter();
            return;
        }

        grid.innerHTML = _libraryAssets.map(assetCard).join('');
        renderExtensionFilter();
    } catch (error) {
        grid.innerHTML = `<div class="col-12 text-center text-danger py-5"><i class="fa-solid fa-triangle-exclamation fa-2x mb-3"></i><p>Failed to load the library: ${escapeHtml((error as Error).message)}</p></div>`;
    }
}

document.getElementById('library-kind')?.addEventListener('change', applyFilters);
document.getElementById('library-extension')?.addEventListener('change', applyFilters);
document.getElementById('library-search')?.addEventListener('input', applyFilters);

void loadLibrary();
