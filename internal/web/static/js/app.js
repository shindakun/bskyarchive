// Minimal JavaScript for confirmation dialogs
document.addEventListener('DOMContentLoaded', function() {
    // Add confirmation to archive start button
    const archiveStartBtn = document.querySelector('button[hx-post="/archive/start"]');
    if (archiveStartBtn) {
        archiveStartBtn.addEventListener('htmx:confirm', function(e) {
            e.preventDefault();
            if (confirm('Start archiving your posts? This may take several minutes.')) {
                e.detail.issueRequest(true);
            }
        });
    }

    // Add confirmation to any future destructive actions
    document.querySelectorAll('[data-confirm]').forEach(function(el) {
        el.addEventListener('htmx:confirm', function(e) {
            e.preventDefault();
            const message = el.getAttribute('data-confirm');
            if (confirm(message)) {
                e.detail.issueRequest(true);
            }
        });
    });
});
