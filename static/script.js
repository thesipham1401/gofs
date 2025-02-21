const path = window.location.pathname;

class UploadFile {
    /**
     * 
     * @param {string} dir 
     * @param {File} file 
     */
    constructor(dir, file) {
        this.dir = dir;
        this.file = file;
    }
}

/**
 * @param {DragEvent} ev 
 */
async function dropHandler(ev) {
    // Prevent default behavior (Prevent file from being opened)
    ev.preventDefault();

    if (ev.dataTransfer.items) {
        // Use DataTransferItemList interface to access the file(s)
        const futuresFile = [...ev.dataTransfer.items].filter((item) => item.kind === "file").map(async (item) => {
            const entry = item.webkitGetAsEntry();
            if (entry.isFile) {
                const file = await getFile(entry);
                return [new UploadFile("", file)];
            } else if (entry.isDirectory) {
                let queue = []
                const uploadFiles = []
                queue = queue.concat(await getSubEntries(entry));
                queue = queue.map((e) => {
                    return {
                        dir: entry.name,
                        entry: e,
                    }
                });
                while (queue.length > 0) {
                    firstEntry = queue[0];
                    queue = queue.slice(1);
                    if (firstEntry.entry.isFile) {
                        const file = await getFile(firstEntry.entry);
                        uploadFiles.push(new UploadFile(firstEntry.dir, file));
                    } else if (firstEntry.entry.isDirectory) {
                        const subEntries = await getSubEntries(firstEntry.entry);
                        for (e of subEntries) {
                            queue.push({ dir: firstEntry.dir + "/" + firstEntry.entry.name, entry: e })
                        }
                    }
                }
                return uploadFiles;
            } else {
                return [];
            }
        });
        const files = (await Promise.all(futuresFile)).flat();
        await Promise.all(files.map(uploadFile));
    } else {
        // Use DataTransfer interface to access the file(s)
        const futures = [...ev.dataTransfer.files].map(uploadFile);
        await Promise.all(futures);
    }
    window.location.reload();
}

/**
 * @param {DragEvent} ev
 */
function dragOverHandler(ev) {
    ev.preventDefault();
}

/**
 * 
 * @param {UploadFile} file
 */
async function uploadFile(file) {
    let safePath = path;
    if (path.startsWith("/")) {
        safePath = path.substring(1);
    }
    let uploadPath;
    if (file.dir === "") {
        uploadPath = safePath;
    } else {
        uploadPath = safePath + "/" + file.dir;
    }
    const formData = new FormData()
    formData.append("path", uploadPath);
    formData.append("submit", "Upload");
    formData.append("files", file.file);
    await fetch('/upload', { method: "POST", body: formData });
}

/**
 * 
 * @param {FileSystemDirectoryEntry} entry
 * @returns {Promise<FileSystemEntry>}
 */
async function getSubEntries(entry) {
    const reader = entry.createReader();
    const entriesPromise = new Promise((resolve, reject) => {
        reader.readEntries(resolve, reject);
    });
    return await entriesPromise;
}

/**
 * 
 * @param {FileSystemFileEntry} entry
 * @returns {Promise<File>}
 */
async function getFile(entry) {
    const promise = new Promise((resolve, reject) => {
        entry.file(resolve, reject);
    });
    return await promise;
}