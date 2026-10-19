interface LibAsset {
    id: number;
    hash: string;
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

interface LibExtension {
    id: number;
    name: string;
}

interface LibTrack {
    id: string;
    name: string;
    country?: string;
}

let _libAssets: LibAsset[] = [];
let _libExtensions: LibExtension[] = [];
let _libTracks: LibTrack[] = [];
let _libSearchTimer: number | undefined;

function libEsc(value: unknown): string {
    return String(value == null ? '' : value).replace(/[&<>"']/g, (m) => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;',
    }[m] as string));
}

function libToast(message: string, type: 'success' | 'error' | 'info' = 'info'): void {
    const fn = (window as any).showToast;
    if (typeof fn === 'function') {
        fn(message, type);
    } else if (type === 'error') {
        alert(message);
    }
}

async function libApi<T>(url: string, method = 'GET', body?: unknown): Promise<T> {
    const opts: RequestInit = { method };
    if (body !== undefined) {
        opts.headers = { 'Content-Type': 'application/json' };
        opts.body = JSON.stringify(body);
    }
    const res = await fetch(url, opts);
    const text = await res.text();
    let data: any = undefined;
    if (text) {
        try {
            data = JSON.parse(text);
        } catch {
            data = text;
        }
    }
    if (!res.ok) {
        throw new Error((data && data.error) || res.statusText || 'Request failed');
    }
    return data as T;
}

function libFormatBytes(bytes: number): string {
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

async function libLoadMeta(): Promise<void> {
    const [extensions, tracks] = await Promise.all([
        libApi<LibExtension[]>('/api/extensions').catch(() => [] as LibExtension[]),
        libApi<LibTrack[]>('/api/tracks').catch(() => [] as LibTrack[]),
    ]);
    _libExtensions = Array.isArray(extensions) ? extensions : [];
    _libTracks = Array.isArray(tracks) ? tracks : [];

    const filter = document.getElementById('library-filter-extension') as HTMLSelectElement | null;
    if (filter) {
        const current = filter.value;
        filter.innerHTML = '<option value="">All extensions</option>' +
            _libExtensions.map((e) => `<option value="${e.id}">${libEsc(e.name)}</option>`).join('');
        filter.value = current;
    }

    const modalExtension = document.getElementById('asset-extension') as HTMLSelectElement | null;
    if (modalExtension) {
        const current = modalExtension.value;
        modalExtension.innerHTML = '<option value="0">None</option>' +
            _libExtensions.map((e) => `<option value="${e.id}">${libEsc(e.name)}</option>`).join('');
        modalExtension.value = current;
    }

    const modalTrack = document.getElementById('asset-track') as HTMLSelectElement | null;
    if (modalTrack) {
        const current = modalTrack.value;
        modalTrack.innerHTML = '<option value="">None</option>' +
            _libTracks.map((t) => `<option value="${libEsc(t.id)}">${libEsc(t.name)}${t.country ? ` (${libEsc(t.country)})` : ''}</option>`).join('');
        modalTrack.value = current;
    }
}

function libKindBadge(kind: string): string {
    if (kind === 'map') return '<span class="badge bg-primary">Map</span>';
    if (kind === 'extension') return '<span class="badge bg-danger">Extension</span>';
    return '<span class="badge bg-secondary">Other</span>';
}

function libPreview(asset: LibAsset): string {
    if (asset.thumbnail_url) {
        return `<a href="${libEsc(asset.url)}" target="_blank" rel="noopener"><img src="${libEsc(asset.thumbnail_url)}" alt="" loading="lazy" class="rounded border" style="width:44px;height:44px;object-fit:cover"></a>`;
    }
    const icon = asset.mime_type === 'application/pdf' ? 'fa-file-pdf text-danger' : 'fa-file text-secondary';
    return `<a href="${libEsc(asset.url)}" target="_blank" rel="noopener" class="btn btn-sm btn-outline-secondary"><i class="fa-solid ${icon}"></i></a>`;
}

function libRenderList(): void {
    const tbody = document.getElementById('library-list');
    if (!tbody) return;

    if (!_libAssets.length) {
        tbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted py-5"><i class="fa-solid fa-folder-open fa-2x mb-2 d-block opacity-50"></i>No assets match these filters yet.</td></tr>';
        return;
    }

    tbody.innerHTML = _libAssets.map((asset) => {
        const associations: string[] = [];
        if (asset.extension_name) associations.push(`<span class="badge bg-light text-dark border"><i class="fa-solid fa-box-open me-1"></i>${libEsc(asset.extension_name)}</span>`);
        if (asset.track_name) associations.push(`<span class="badge bg-light text-dark border"><i class="fa-solid fa-route me-1"></i>${libEsc(asset.track_name)}</span>`);
        const association = associations.length ? associations.join(' ') : '<span class="text-muted">—</span>';
        const description = asset.description
            ? `<div class="small text-muted text-truncate" style="max-width:340px">${libEsc(asset.description)}</div>`
            : '';

        return `<tr>
            <td class="ps-4">${libPreview(asset)}</td>
            <td>
                <div class="fw-bold">${libEsc(asset.title)}</div>
                <div class="small text-muted text-truncate" style="max-width:340px">${libEsc(asset.original_name)}</div>
                ${description}
            </td>
            <td>${libKindBadge(asset.kind)}</td>
            <td>${association}</td>
            <td class="text-end small text-muted text-nowrap">${libFormatBytes(asset.size_bytes)}</td>
            <td class="small text-muted text-nowrap">${libEsc(asset.created_at.slice(0, 10))}</td>
            <td class="text-end pe-4 text-nowrap">
                <a class="btn btn-sm btn-outline-secondary" href="${libEsc(asset.url)}" target="_blank" rel="noopener" title="Preview"><i class="fa-solid fa-up-right-from-square"></i></a>
                <a class="btn btn-sm btn-outline-secondary" href="/api/assets/${asset.id}/download" title="Download"><i class="fa-solid fa-download"></i></a>
                <button class="btn btn-sm btn-outline-primary" onclick="libEditAsset(${asset.id})" title="Edit"><i class="fa-solid fa-pen"></i></button>
                <button class="btn btn-sm btn-outline-danger" onclick="libDeleteAsset(${asset.id})" title="Delete"><i class="fa-solid fa-trash"></i></button>
            </td>
        </tr>`;
    }).join('');
}

async function libLoadLibrary(): Promise<void> {
    const tbody = document.getElementById('library-list');
    if (!tbody) return;

    tbody.innerHTML = '<tr><td colspan="7" class="text-center text-muted py-4"><i class="fa-solid fa-spinner fa-spin me-2"></i>Loading library...</td></tr>';

    try {
        await libLoadMeta();

        const params = new URLSearchParams();
        const kind = (document.getElementById('library-filter-kind') as HTMLSelectElement | null)?.value || '';
        const extensionID = (document.getElementById('library-filter-extension') as HTMLSelectElement | null)?.value || '';
        const search = (document.getElementById('library-search') as HTMLInputElement | null)?.value.trim() || '';
        if (kind) params.set('kind', kind);
        if (extensionID) params.set('extension_id', extensionID);
        if (search) params.set('q', search);

        const assets = await libApi<LibAsset[]>('/api/assets?' + params.toString());
        _libAssets = Array.isArray(assets) ? assets : [];
        libRenderList();
    } catch (error) {
        tbody.innerHTML = `<tr><td colspan="7" class="text-center text-danger py-4">Failed to load library: ${libEsc((error as Error).message)}</td></tr>`;
    }
}

function libLoadLibraryDebounced(): void {
    if (_libSearchTimer !== undefined) window.clearTimeout(_libSearchTimer);
    _libSearchTimer = window.setTimeout(() => { void libLoadLibrary(); }, 250);
}

function libClearFilters(): void {
    const kind = document.getElementById('library-filter-kind') as HTMLSelectElement | null;
    const extension = document.getElementById('library-filter-extension') as HTMLSelectElement | null;
    const search = document.getElementById('library-search') as HTMLInputElement | null;
    if (kind) kind.value = '';
    if (extension) extension.value = '';
    if (search) search.value = '';
    void libLoadLibrary();
}

function libSyncAssetForm(): void {
    const kind = (document.getElementById('asset-kind') as HTMLSelectElement | null)?.value || 'other';
    const wrap = document.getElementById('asset-background-wrap');
    if (wrap) wrap.style.display = kind === 'map' ? '' : 'none';

    const checkbox = document.getElementById('asset-set-background') as HTMLInputElement | null;
    if (checkbox && kind !== 'map') checkbox.checked = false;
}

async function libOpenAssetModal(id?: number, defaults?: { kind?: string; extension_id?: number; track_id?: string }): Promise<void> {
    try {
        await libLoadMeta();
    } catch {
        // The modal still works without the optional association lists.
    }

    const isEdit = typeof id === 'number' && id > 0;
    let asset = isEdit ? _libAssets.find((a) => a.id === id) : undefined;
    if (isEdit && !asset) {
        try {
            const all = await libApi<LibAsset[]>('/api/assets?limit=500');
            asset = all.find((a) => a.id === id);
        } catch {
            asset = undefined;
        }
    }

    const idInput = document.getElementById('asset-id') as HTMLInputElement | null;
    const titleInput = document.getElementById('asset-title') as HTMLInputElement | null;
    const descriptionInput = document.getElementById('asset-description') as HTMLTextAreaElement | null;
    const kindSelect = document.getElementById('asset-kind') as HTMLSelectElement | null;
    const extensionSelect = document.getElementById('asset-extension') as HTMLSelectElement | null;
    const trackSelect = document.getElementById('asset-track') as HTMLSelectElement | null;
    const background = document.getElementById('asset-set-background') as HTMLInputElement | null;
    const fileInput = document.getElementById('asset-file') as HTMLInputElement | null;
    const fileHelp = document.getElementById('asset-file-help');
    const saveLabel = document.getElementById('asset-save-label');
    const modalTitle = document.getElementById('assetModalTitle');

    if (idInput) idInput.value = asset ? String(asset.id) : '0';
    if (titleInput) titleInput.value = asset ? asset.title : '';
    if (descriptionInput) descriptionInput.value = asset ? asset.description : '';
    if (kindSelect) kindSelect.value = asset ? asset.kind : (defaults?.kind || 'map');
    if (extensionSelect) extensionSelect.value = String(asset ? asset.extension_id : (defaults?.extension_id || 0));
    if (trackSelect) trackSelect.value = asset ? asset.track_id : (defaults?.track_id || '');
    if (background) background.checked = false;
    if (fileInput) {
        fileInput.value = '';
        fileInput.disabled = isEdit;
    }
    if (fileHelp) {
        fileHelp.textContent = isEdit
            ? 'Editing metadata only. Delete the asset and upload again to replace the file.'
            : 'PDF, PNG, JPG, WebP, or GIF. Maximum 25 MB. Identical files are not stored twice.';
    }
    if (saveLabel) saveLabel.textContent = isEdit ? 'Save' : 'Upload';
    if (modalTitle) modalTitle.textContent = isEdit ? 'Edit Asset' : 'Upload Asset';

    libSyncAssetForm();
    const modalElement = document.getElementById('assetModal');
    if (modalElement) {
        (window as any).bootstrap.Modal.getOrCreateInstance(modalElement).show();
    }
}

function libOpenAssetModalForExtension(extensionID: number): void {
    void libOpenAssetModal(undefined, { kind: 'extension', extension_id: extensionID });
}

function libEditAsset(id: number): void {
    void libOpenAssetModal(id);
}

async function libSaveAsset(): Promise<void> {
    const id = parseInt((document.getElementById('asset-id') as HTMLInputElement | null)?.value || '0', 10) || 0;
    const title = (document.getElementById('asset-title') as HTMLInputElement | null)?.value.trim() || '';
    const description = (document.getElementById('asset-description') as HTMLTextAreaElement | null)?.value.trim() || '';
    const kind = (document.getElementById('asset-kind') as HTMLSelectElement | null)?.value || 'other';
    const extensionID = parseInt((document.getElementById('asset-extension') as HTMLSelectElement | null)?.value || '0', 10) || 0;
    const trackID = (document.getElementById('asset-track') as HTMLSelectElement | null)?.value || '';
    const setBackground = (document.getElementById('asset-set-background') as HTMLInputElement | null)?.checked || false;

    if (!title) {
        libToast('Title is required', 'error');
        return;
    }
    if (kind === 'extension' && !extensionID) {
        libToast('Choose an extension for an extension file', 'error');
        return;
    }
    if (setBackground && !trackID) {
        libToast('Choose a track first, or turn off the background option', 'error');
        return;
    }

    try {
        if (id > 0) {
            await libApi<LibAsset>('/api/assets', 'PUT', {
                id,
                title,
                description,
                kind,
                extension_id: extensionID,
                track_id: trackID,
            });
            libToast('Asset updated', 'success');
        } else {
            const fileInput = document.getElementById('asset-file') as HTMLInputElement | null;
            const file = fileInput?.files?.[0];
            if (!file) {
                libToast('Choose a file to upload', 'error');
                return;
            }

            const fd = new FormData();
            fd.append('file', file);
            fd.append('title', title);
            fd.append('description', description);
            fd.append('kind', kind);
            fd.append('extension_id', String(extensionID));
            fd.append('track_id', trackID);
            if (setBackground) fd.append('set_as_map_background', '1');

            const res = await fetch('/api/assets', { method: 'POST', body: fd });
            const data = await res.json().catch(() => ({}));
            if (!res.ok) throw new Error(data.error || res.statusText || 'Upload failed');
            if (data.duplicate) {
                libToast('That file is already in the library; the existing copy was reused', 'info');
            } else {
                libToast('Asset uploaded', 'success');
            }
        }

        const modalElement = document.getElementById('assetModal');
        if (modalElement) {
            (window as any).bootstrap.Modal.getInstance(modalElement)?.hide();
        }

        if (document.getElementById('library-list')) {
            await libLoadLibrary();
        }
        const currentExtension = (window as any)._currentExtensionDetailId;
        if (currentExtension && typeof (window as any).loadExtensionDetail === 'function') {
            (window as any).loadExtensionDetail(currentExtension);
        }
    } catch (error) {
        libToast('Failed: ' + (error as Error).message, 'error');
    }
}

async function libDeleteAsset(id: number, skipConfirm?: boolean): Promise<void> {
    if (skipConfirm !== true && !confirm('Delete this asset and its stored file?')) return;
    try {
        await libApi<unknown>('/api/assets?id=' + id, 'DELETE');
        libToast('Asset deleted', 'success');
        _libAssets = _libAssets.filter((asset) => asset.id !== id);
        if (document.getElementById('library-list')) {
            libRenderList();
        }
        const currentExtension = (window as any)._currentExtensionDetailId;
        if (currentExtension && typeof (window as any).loadExtensionDetail === 'function') {
            (window as any).loadExtensionDetail(currentExtension);
        }
    } catch (error) {
        libToast('Failed: ' + (error as Error).message, 'error');
    }
}

(window as any).libLoadLibrary = libLoadLibrary;
(window as any).libLoadLibraryDebounced = libLoadLibraryDebounced;
(window as any).libClearFilters = libClearFilters;
(window as any).libOpenAssetModal = libOpenAssetModal;
(window as any).libOpenAssetModalForExtension = libOpenAssetModalForExtension;
(window as any).libEditAsset = libEditAsset;
(window as any).libSaveAsset = libSaveAsset;
(window as any).libDeleteAsset = libDeleteAsset;
(window as any).libSyncAssetForm = libSyncAssetForm;
(window as any).libFormatBytes = libFormatBytes;
