const API_BASE = '/api';

async function fetchProject() {
    try {
        const res = await fetch(`${API_BASE}/project`);
        const data = await res.json();
        document.getElementById('project-name').textContent = data.name || 'Unknown Project';
    } catch (e) {
        console.error(e);
    }
}

async function fetchJobs() {
    try {
        const res = await fetch(`${API_BASE}/jobs`);
        const data = await res.json();
        document.getElementById('stat-pending').textContent = data.pending || 0;
        document.getElementById('stat-processing').textContent = data.processing || 0;
        document.getElementById('stat-completed').textContent = data.completed || 0;
        document.getElementById('stat-failed').textContent = data.failed || 0;
    } catch (e) {
        console.error(e);
    }
}

async function fetchFiles() {
    try {
        const res = await fetch(`${API_BASE}/files`);
        const files = await res.json();
        const list = document.getElementById('file-list');
        list.innerHTML = '';
        
        files.forEach(f => {
            const el = document.createElement('div');
            el.className = 'file-item';
            el.textContent = f.path;
            el.onclick = () => selectFile(f.id, f.path, el);
            list.appendChild(el);
        });
    } catch (e) {
        console.error(e);
    }
}

async function selectFile(id, path, el) {
    document.querySelectorAll('.file-item').forEach(n => n.classList.remove('active'));
    el.classList.add('active');
    
    document.getElementById('current-file-path').textContent = path;
    document.getElementById('file-summary').textContent = "Loading intelligence...";
    document.getElementById('feature-list').innerHTML = `<div class="pulse-ring" style="margin:2rem auto"></div>`;

    try {
        const res = await fetch(`${API_BASE}/files/${id}`);
        const data = await res.json();
        
        document.getElementById('file-summary').textContent = data.summary || "No summary available. File might not be processed yet.";
        
        const featList = document.getElementById('feature-list');
        featList.innerHTML = '';
        if (data.features && data.features.length > 0) {
            data.features.forEach(feat => {
                const card = document.createElement('div');
                card.className = 'feature-card';
                card.innerHTML = `<h4>${feat.name}</h4><p>${feat.description}</p>`;
                featList.appendChild(card);
            });
        } else {
            featList.innerHTML = `<p class="text-secondary">No features extracted yet.</p>`;
        }
    } catch (e) {
        console.error(e);
        document.getElementById('file-summary').textContent = "Error loading intelligence.";
    }
}

function pollJobs() {
    setInterval(fetchJobs, 2000);
}

// Init
fetchProject();
fetchFiles();
fetchJobs();
pollJobs();
