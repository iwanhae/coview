// Enhanced drag and drop upload with preview, progress, search, and lazy loading
document.addEventListener('DOMContentLoaded', function() {
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file-input');
    const uploadForm = document.getElementById('upload-form');
    const statusDiv = document.getElementById('upload-status');
    const filePreview = document.getElementById('file-preview');
    const searchInput = document.getElementById('search-input');
    const zipsContainer = document.getElementById('zips-container');
    const noResults = document.getElementById('no-results');
    let selectedFiles = [];

    // Prevent default drag behaviors
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });

    // Highlight drop zone
    ['dragenter', 'dragover'].forEach(eventName => {
        dropZone.addEventListener(eventName, highlight, false);
    });
    ['dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, unhighlight, false);
    });

    // Handle drop
    dropZone.addEventListener('drop', handleDrop, false);

    // Handle file input change
    fileInput.addEventListener('change', (e) => {
        handleFiles(e.target.files);
        e.target.value = ''; // Reset for next selection
    });

    // Search functionality
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            const query = e.target.value.toLowerCase();
            const cards = zipsContainer.querySelectorAll('.zip-card');
            let visibleCount = 0;

            cards.forEach(card => {
                const name = card.dataset.name.toLowerCase();
                if (name.includes(query)) {
                    card.style.display = 'block';
                    visibleCount++;
                } else {
                    card.style.display = 'none';
                }
            });

            noResults.classList.toggle('hidden', visibleCount > 0);
            if (visibleCount === 0 && query) {
                noResults.classList.remove('hidden');
            } else if (query === '') {
                noResults.classList.add('hidden');
            }
        });
    }

    // Lazy loading for thumbnails
    if ('IntersectionObserver' in window) {
        const imageObserver = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src || img.src;
                    img.classList.remove('lazy');
                    imageObserver.unobserve(img);
                }
            });
        });

        document.querySelectorAll('img.lazy').forEach(img => {
            img.dataset.src = img.src;
            img.src = ''; // Placeholder
            imageObserver.observe(img);
        });
    }

    function preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }

    function highlight(e) {
        dropZone.classList.add('border-blue-400', 'bg-blue-50');
    }

    function unhighlight(e) {
        dropZone.classList.remove('border-blue-400', 'bg-blue-50');
    }

    function handleDrop(e) {
        const dt = e.dataTransfer;
        const files = dt.files;
        handleFiles(files);
    }

    function handleFiles(filesObj) {
        const files = Array.from(filesObj).filter(file => 
            file.name.toLowerCase().endsWith('.zip')
        );

        if (files.length === 0) {
            showStatus('No ZIP files found in selection', 'error');
            return;
        }

        selectedFiles = files;
        displayPreview(files);
    }

    function displayPreview(files) {
        filePreview.innerHTML = '';
        filePreview.classList.remove('hidden');

        files.forEach((file, index) => {
            const size = (file.size / 1024 / 1024).toFixed(2) + ' MB';
            const div = document.createElement('div');
            div.className = 'border border-gray-300 rounded p-3 bg-gray-50';
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
        const uploadBtn = document.createElement('button');
        uploadBtn.textContent = `Upload ${files.length} Selected File(s)`;
        uploadBtn.className = 'mt-4 bg-green-500 hover:bg-green-700 text-white font-bold py-2 px-4 rounded w-full';
        uploadBtn.onclick = () => uploadFiles(selectedFiles);
        filePreview.appendChild(uploadBtn);
    }

    window.removeFile = function(index) {
        selectedFiles.splice(index, 1);
        if (selectedFiles.length === 0) {
            filePreview.classList.add('hidden');
        } else {
            displayPreview(selectedFiles);
        }
    };

    function uploadFiles(files) {
        if (files.length === 0) return;

        showStatus(`Uploading ${files.length} file(s)...`, 'info');

        files.forEach((file, index) => {
            const formData = new FormData();
            formData.append('files', file);

            const xhr = new XMLHttpRequest();
            xhr.open('POST', '/upload', true);

            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    const percent = (e.loaded / e.total) * 100;
                    document.getElementById(`progress-${index}`).style.width = percent + '%';
                }
            });

            xhr.onload = function() {
                if (xhr.status === 200) {
                    showStatus(`Successfully uploaded ${file.name}`, 'success');
                } else {
                    showStatus(`Upload failed for ${file.name}: ${xhr.responseText}`, 'error');
                }
            };

            xhr.onerror = function() {
                showStatus(`Network error uploading ${file.name}`, 'error');
            };

            xhr.send(formData);
        });

        // Reload after all uploads (simple, or use Promise.all for parallel)
        setTimeout(() => {
            if (confirm('Upload complete. Reload page to see new comics?')) {
                window.location.reload();
            }
            filePreview.classList.add('hidden');
            selectedFiles = [];
        }, 2000);
    }

    // Handle form submit (for compatibility)
    uploadForm.addEventListener('submit', function(e) {
        e.preventDefault();
        if (selectedFiles.length > 0) {
            uploadFiles(selectedFiles);
        }
    });

    function showStatus(message, type) {
        statusDiv.textContent = message;
        statusDiv.className = `mt-4 p-4 rounded ${type === 'info' ? 'bg-blue-100 text-blue-800' : type === 'success' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'} ${statusDiv.className.includes('hidden') ? '' : 'block'}`;
        statusDiv.classList.remove('hidden');

        if (type === 'success') {
            setTimeout(() => {
                statusDiv.classList.add('hidden');
            }, 3000);
        }
    }

    // Click to open file dialog
    dropZone.addEventListener('click', () => {
        fileInput.click();
    });
});
