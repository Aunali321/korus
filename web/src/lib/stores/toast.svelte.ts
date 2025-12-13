export interface Toast {
    id: string;
    message: string;
    type: 'success' | 'error' | 'info';
}

function createToastStore() {
    let toasts = $state<Toast[]>([]);

    function add(message: string, type: Toast['type'] = 'info') {
        const id = Math.random().toString(36).slice(2);
        toasts = [...toasts, { id, message, type }];

        setTimeout(() => {
            remove(id);
        }, 3000);
    }

    function remove(id: string) {
        toasts = toasts.filter((t) => t.id !== id);
    }

    function success(message: string) {
        add(message, 'success');
    }

    function error(message: string) {
        add(message, 'error');
    }

    function info(message: string) {
        add(message, 'info');
    }

    return {
        get toasts() { return toasts; },
        add,
        remove,
        success,
        error,
        info
    };
}

export const toast = createToastStore();
