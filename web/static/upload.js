// Enhanced drag and drop upload with preview, progress, search, and lazy loading
document.addEventListener("DOMContentLoaded", function () {
    const dropZone = document.getElementById("drop-zone");
    const fileInput = document.getElementById("file-input");
    const uploadForm = document.getElementById("upload-form");
    const statusDiv = document.getElementById("upload-status");
    const filePreview = document.getElementById("file-preview");
    const searchInput = document.getElementById("search-input");
    const zipsContainer = document.getElementById("zips-container");
    const noResults = document.getElementById("no-results");
    const sortSelect = document.getElementById("sort-by");
    let selectedFiles = [];

    // Natural sort comparison function (handles numbers in strings correctly)
    function naturalCompare(a, b) {
        const ax = [];
        const bx = [];

        a.replace(/(\d+)|(\D+)/g, function (_, num, str) {
            ax.push([num || 0, str || ""]);
        });
        b.replace(/(\d+)|(\D+)/g, function (_, num, str) {
            bx.push([num || 0, str || ""]);
        });

        while (ax.length && bx.length) {
            const an = ax.shift();
            const bn = bx.shift();
            const nn = an[0] - bn[0] || an[1].localeCompare(bn[1]);
            if (nn) return nn;
        }

        return ax.length - bx.length;
    }

    function sortComics(sortValue) {
        const cards = Array.from(zipsContainer.querySelectorAll(".zip-card"));

        // Store visibility state
        const visibilityMap = new Map();
        cards.forEach((card) => {
            visibilityMap.set(card, card.style.display);
        });

        // Sort all cards
        cards.sort((a, b) => {
            const [sortType, order] = sortValue.split("-");
            let comparison = 0;

            switch (sortType) {
                case "name":
                    const nameA = (a.dataset.name || "").toLowerCase();
                    const nameB = (b.dataset.name || "").toLowerCase();
                    comparison = naturalCompare(nameA, nameB);
                    break;
                case "date":
                    const dateA = parseInt(a.dataset.modtime || "0", 10);
                    const dateB = parseInt(b.dataset.modtime || "0", 10);
                    comparison = dateA - dateB;
                    break;
                case "size":
                    const sizeA = parseInt(a.dataset.size || "0", 10);
                    const sizeB = parseInt(b.dataset.size || "0", 10);
                    comparison = sizeA - sizeB;
                    break;
            }

            return order === "desc" ? -comparison : comparison;
        });

        // Re-append sorted cards and restore visibility
        cards.forEach((card) => {
            const originalDisplay = visibilityMap.get(card);
            if (originalDisplay === "none") {
                card.style.display = "none";
            } else {
                card.style.display = "";
            }
            zipsContainer.appendChild(card);
        });
    }

    // Apply saved sort immediately to prevent double render
    const savedSort = localStorage.getItem("comicSortPreference");
    if (savedSort && sortSelect) {
        sortSelect.value = savedSort;
        // Apply sort synchronously before anything else renders
        sortComics(savedSort);
    }

    // Prevent default drag behaviors
    ["dragenter", "dragover", "dragleave", "drop"].forEach((eventName) => {
        dropZone.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });

    // Highlight drop zone
    ["dragenter", "dragover"].forEach((eventName) => {
        dropZone.addEventListener(eventName, highlight, false);
    });
    ["dragleave", "drop"].forEach((eventName) => {
        dropZone.addEventListener(eventName, unhighlight, false);
    });

    // Handle drop
    dropZone.addEventListener("drop", handleDrop, false);

    // Handle file input change
    fileInput.addEventListener("change", (e) => {
        handleFiles(e.target.files);
        e.target.value = ""; // Reset for next selection
    });

    // Search functionality
    if (searchInput) {
        searchInput.addEventListener("input", (e) => {
            const query = e.target.value.toLowerCase();
            const cards = zipsContainer.querySelectorAll(".zip-card");
            let visibleCount = 0;

            cards.forEach((card) => {
                const name = card.dataset.name.toLowerCase();
                if (name.includes(query)) {
                    card.style.display = "";
                    visibleCount++;
                } else {
                    card.style.display = "none";
                }
            });

            noResults.classList.toggle("hidden", visibleCount > 0);
            if (visibleCount === 0 && query) {
                noResults.classList.remove("hidden");
            } else if (query === "") {
                noResults.classList.add("hidden");
            }

            // Re-apply current sort after filtering
            if (sortSelect && sortSelect.value) {
                sortComics(sortSelect.value);
            }
        });
    }

    // Sort functionality
    if (sortSelect) {
        sortSelect.addEventListener("change", (e) => {
            const sortValue = e.target.value;
            sortComics(sortValue);
            // Save sort preference to localStorage
            localStorage.setItem("comicSortPreference", sortValue);
        });
    }

    // Lazy loading for thumbnails
    if ("IntersectionObserver" in window) {
        const imageObserver = new IntersectionObserver((entries) => {
            entries.forEach((entry) => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src || img.src;
                    img.classList.remove("lazy");
                    imageObserver.unobserve(img);
                }
            });
        });

        document.querySelectorAll("img.lazy").forEach((img) => {
            img.dataset.src = img.src;
            img.src = ""; // Placeholder
            imageObserver.observe(img);
        });
    }

    function preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }

    function highlight(e) {
        dropZone.classList.add("border-blue-400", "bg-blue-50");
    }

    function unhighlight(e) {
        dropZone.classList.remove("border-blue-400", "bg-blue-50");
    }

    function handleDrop(e) {
        const dt = e.dataTransfer;
        const files = dt.files;
        handleFiles(files);
    }

    function handleFiles(filesObj) {
        const files = Array.from(filesObj).filter((file) =>
            file.name.toLowerCase().endsWith(".zip")
        );

        if (files.length === 0) {
            showStatus("No ZIP files found in selection", "error");
            return;
        }

        selectedFiles = files;
        displayPreview(files);
    }

    function displayPreview(files) {
        filePreview.innerHTML = "";
        filePreview.classList.remove("hidden");

        files.forEach((file, index) => {
            const size = (file.size / 1024 / 1024).toFixed(2) + " MB";
            const div = document.createElement("div");
            div.className = "border border-gray-300 rounded p-3 bg-gray-50";
            div.innerHTML = `
                <div class="flex justify-between items-center mb-2">
                    <span class="font-medium">${file.name}</span>
                    <span class="text-sm text-gray-500">${size}</span>
                </div>
                <div class="flex justify-end">
                    <button onclick="removeFile(${index})" class="text-red-500 hover:text-red-700 text-sm">Remove</button>
                </div>
                <div class="mt-2">
                    <div class="w-full bg-gray-200 rounded-full h-2">
                        <div id="progress-${index}" class="bg-blue-600 h-2 rounded-full transition-all duration-300" style="width: 0%"></div>
                    </div>
                </div>
            `;
            filePreview.appendChild(div);
        });

        // Add upload button
        const uploadBtn = document.createElement("button");
        uploadBtn.type = "button";
        uploadBtn.textContent = `Upload ${files.length} Selected File(s)`;
        uploadBtn.className =
            "mt-4 bg-green-500 hover:bg-green-700 text-white font-bold py-2 px-4 rounded w-full";
        uploadBtn.onclick = () => uploadFiles(selectedFiles);
        filePreview.appendChild(uploadBtn);
    }

    window.removeFile = function (index) {
        selectedFiles.splice(index, 1);
        if (selectedFiles.length === 0) {
            filePreview.classList.add("hidden");
        } else {
            displayPreview(selectedFiles);
        }
    };

    async function uploadFiles(files) {
        if (files.length === 0) return;

        // Disable upload button during upload
        const uploadBtn = filePreview.querySelector("button");
        uploadBtn.disabled = true;
        uploadBtn.textContent = "Uploading...";

        showStatus(`Starting upload of ${files.length} file(s)...`, "info");

        let successCount = 0;
        let errorCount = 0;
        const errorMessages = [];

        // Add overall progress bar
        const overallProgressDiv = document.createElement("div");
        overallProgressDiv.className = "mt-4";
        overallProgressDiv.innerHTML = `
            <div class="text-sm text-gray-600 mb-1">Overall Progress</div>
            <div class="w-full bg-gray-200 rounded-full h-3">
                <div id="overall-progress" class="bg-blue-600 h-3 rounded-full transition-all duration-300" style="width: 0%"></div>
            </div>
        `;
        filePreview.appendChild(overallProgressDiv);

        for (let i = 0; i < files.length; i++) {
            const file = files[i];
            const index = i;

            // Update current file status
            showStatus(
                `Uploading ${index + 1}/${files.length}: ${file.name}`,
                "info"
            );

            try {
                await uploadSingleFile(file, index);
                successCount++;
            } catch (error) {
                errorCount++;
                errorMessages.push(`Failed to upload ${file.name}: ${error.message}`);
                showStatus(`Failed to upload ${file.name}`, "error");
            }

            // Update overall progress
            const overallPercent = ((index + 1) / files.length) * 100;
            document.getElementById("overall-progress").style.width =
                overallPercent + "%";
        }

        // Remove overall progress bar
        overallProgressDiv.remove();

        // Show final result
        if (errorCount === 0) {
            showStatus(
                `Successfully uploaded all ${successCount} file(s)`,
                "success"
            );
        } else {
            showStatus(
                `Upload completed with ${successCount} success and ${errorCount} error(s). ${errorMessages.join(
                    "; "
                )}`,
                "error"
            );
        }

        // Re-enable upload button and update text
        uploadBtn.disabled = false;
        uploadBtn.textContent = `Upload ${selectedFiles.length} Selected File(s)`;

        // Reload after successful uploads
        if (
            successCount > 0 &&
            confirm("Upload complete. Reload page to see new comics?")
        ) {
            window.location.reload();
        }

        // Reset if all files are done
        if (selectedFiles.length === 0) {
            filePreview.classList.add("hidden");
        }
    }

    function uploadSingleFile(file, index) {
        return new Promise((resolve, reject) => {
            const formData = new FormData();
            formData.append("files", file);

            const xhr = new XMLHttpRequest();
            xhr.open("POST", "/upload", true);

            xhr.upload.addEventListener("progress", (e) => {
                if (e.lengthComputable) {
                    const percent = (e.loaded / e.total) * 100;
                    document.getElementById(`progress-${index}`).style.width =
                        percent + "%";
                }
            });

            xhr.onload = function () {
                if (xhr.status === 200) {
                    resolve();
                } else {
                    reject(new Error(xhr.responseText || `HTTP ${xhr.status}`));
                }
            };

            xhr.onerror = function () {
                reject(new Error("Network error"));
            };

            xhr.send(formData);
        });
    }

    function showStatus(message, type) {
        statusDiv.textContent = message;
        statusDiv.className = `mt-4 p-4 rounded ${type === "info"
            ? "bg-blue-100 text-blue-800"
            : type === "success"
                ? "bg-green-100 text-green-800"
                : "bg-red-100 text-red-800"
            } ${statusDiv.className.includes("hidden") ? "" : "block"}`;
        statusDiv.classList.remove("hidden");

        if (type === "success") {
            setTimeout(() => {
                statusDiv.classList.add("hidden");
            }, 3000);
        }
    }

    // Drop zone click still opens file dialog for convenience
    dropZone.addEventListener("click", () => {
        fileInput.click();
    });
});
