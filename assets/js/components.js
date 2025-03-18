/**
 * Components.js - JavaScript utilities for the reusable components in the POS system
 */

document.addEventListener('DOMContentLoaded', function() {
    // Initialize all components
    initModals();
    initDropdowns();
    initNotifications();
    initTabs();
});

/**
 * Modal Dialog Functionality
 */
function initModals() {
    // Modal Open Buttons
    const modalOpenButtons = document.querySelectorAll('[data-modal-open]');
    modalOpenButtons.forEach(button => {
        button.addEventListener('click', function() {
            const targetId = this.getAttribute('data-target');
            if (targetId) {
                openModal(targetId);
            }
        });
    });

    // Modal Close Buttons
    const modalCloseButtons = document.querySelectorAll('[data-modal-close]');
    modalCloseButtons.forEach(button => {
        button.addEventListener('click', function() {
            const targetId = this.getAttribute('data-target');
            if (targetId) {
                closeModal(targetId);
            }
        });
    });

    // Close on background click
    const modals = document.querySelectorAll('[role="dialog"]');
    modals.forEach(modal => {
        modal.addEventListener('click', function(event) {
            if (event.target === this) {
                closeModal(this.id);
            }
        });
    });

    // Close on Escape key
    document.addEventListener('keydown', function(event) {
        if (event.key === 'Escape') {
            const openModal = document.querySelector('[role="dialog"]:not(.hidden)');
            if (openModal) {
                closeModal(openModal.id);
            }
        }
    });
}

function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        // Add enter animation classes
        modal.classList.remove('hidden');
        document.body.classList.add('overflow-hidden');
        
        // Set focus on the first focusable element
        setTimeout(() => {
            const focusable = modal.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
            if (focusable.length) {
                focusable[0].focus();
            }
        }, 100);
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        // Add exit animation classes if needed
        modal.classList.add('hidden');
        document.body.classList.remove('overflow-hidden');
    }
}

/**
 * Dropdown Menu Functionality
 */
function initDropdowns() {
    const dropdownButtons = document.querySelectorAll('[data-dropdown-toggle]');
    
    dropdownButtons.forEach(button => {
        button.addEventListener('click', function(e) {
            e.stopPropagation();
            
            const targetId = this.getAttribute('data-dropdown-toggle');
            const dropdownMenu = document.getElementById(targetId);
            
            if (dropdownMenu) {
                dropdownMenu.classList.toggle('hidden');
                
                // Position the dropdown
                const buttonRect = this.getBoundingClientRect();
                dropdownMenu.style.top = (buttonRect.bottom + window.scrollY) + 'px';
                
                // Determine if dropdown should open to the left or right
                const windowWidth = window.innerWidth;
                if (buttonRect.right + 200 > windowWidth) { // Assuming dropdown width is around 200px
                    dropdownMenu.style.right = (windowWidth - buttonRect.right) + 'px';
                    dropdownMenu.style.left = 'auto';
                } else {
                    dropdownMenu.style.left = buttonRect.left + 'px';
                    dropdownMenu.style.right = 'auto';
                }
            }
        });
    });
    
    // Close dropdowns when clicking outside
    document.addEventListener('click', function() {
        const openDropdowns = document.querySelectorAll('.dropdown-menu:not(.hidden)');
        openDropdowns.forEach(dropdown => {
            dropdown.classList.add('hidden');
        });
    });
}

/**
 * Notification System
 */
function initNotifications() {
    // Auto-hide notifications after timeout
    const notifications = document.querySelectorAll('.notification');
    notifications.forEach(notification => {
        // Auto-hide after 5 seconds if it has auto-hide attribute
        if (notification.hasAttribute('data-auto-hide')) {
            setTimeout(() => {
                hideNotification(notification);
            }, 5000);
        }
        
        // Add click handler to close button
        const closeBtn = notification.querySelector('[data-notification-close]');
        if (closeBtn) {
            closeBtn.addEventListener('click', function() {
                hideNotification(notification);
            });
        }
    });
}

function hideNotification(notification) {
    notification.classList.add('opacity-0');
    setTimeout(() => {
        notification.remove();
    }, 300); // Match transition duration
}

// Function to show a new notification
function showNotification(message, type = 'info', autohide = true) {
    // Create notification container if it doesn't exist
    let container = document.getElementById('notification-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'notification-container';
        container.className = 'fixed top-4 right-4 z-50 flex flex-col gap-2';
        document.body.appendChild(container);
    }
    
    // Determine icon and color based on type
    let icon, bgColor;
    switch (type) {
        case 'success':
            icon = 'check-circle';
            bgColor = 'bg-green-100 border-green-500 text-green-800';
            break;
        case 'error':
            icon = 'exclamation-circle';
            bgColor = 'bg-red-100 border-red-500 text-red-800';
            break;
        case 'warning':
            icon = 'exclamation-triangle';
            bgColor = 'bg-yellow-100 border-yellow-500 text-yellow-800';
            break;
        default: // info
            icon = 'info-circle';
            bgColor = 'bg-blue-100 border-blue-500 text-blue-800';
    }
    
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification flex items-center p-4 rounded border-l-4 ${bgColor} transition-opacity duration-300`;
    if (autohide) {
        notification.setAttribute('data-auto-hide', 'true');
    }
    
    notification.innerHTML = `
        <div class="flex-shrink-0 mr-3">
            <i class="fas fa-${icon}"></i>
        </div>
        <div class="flex-grow">${message}</div>
        <button type="button" class="ml-auto focus:outline-none" data-notification-close>
            <i class="fas fa-times"></i>
        </button>
    `;
    
    // Add to container
    container.appendChild(notification);
    
    // Set up auto-hide
    if (autohide) {
        setTimeout(() => {
            hideNotification(notification);
        }, 5000);
    }
    
    // Add close button handler
    const closeBtn = notification.querySelector('[data-notification-close]');
    closeBtn.addEventListener('click', function() {
        hideNotification(notification);
    });
    
    return notification;
}

/**
 * Tabs Functionality
 */
function initTabs() {
    const tabGroups = document.querySelectorAll('[role="tablist"]');
    
    tabGroups.forEach(tabGroup => {
        const tabs = tabGroup.querySelectorAll('[role="tab"]');
        
        tabs.forEach(tab => {
            tab.addEventListener('click', function() {
                // Get the target panel ID
                const target = this.getAttribute('aria-controls');
                const tabContainer = this.closest('[role="tablist"]');
                const tabsWrapper = tabContainer.parentNode;
                
                // Deactivate all tabs in this group
                tabContainer.querySelectorAll('[role="tab"]').forEach(t => {
                    t.setAttribute('aria-selected', 'false');
                    t.classList.remove('border-brand-500', 'text-brand-600');
                    t.classList.add('border-transparent', 'text-gray-500', 'hover:text-gray-700', 'hover:border-gray-300');
                });
                
                // Activate this tab
                this.setAttribute('aria-selected', 'true');
                this.classList.remove('border-transparent', 'text-gray-500', 'hover:text-gray-700', 'hover:border-gray-300');
                this.classList.add('border-brand-500', 'text-brand-600');
                
                // Hide all tab panels
                const panels = tabsWrapper.querySelectorAll('[role="tabpanel"]');
                panels.forEach(panel => {
                    panel.classList.add('hidden');
                });
                
                // Show the target panel
                const targetPanel = document.getElementById(target);
                if (targetPanel) {
                    targetPanel.classList.remove('hidden');
                }
            });
        });
    });
} 