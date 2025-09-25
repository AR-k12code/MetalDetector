// Handle geolocation requests from the service worker
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.type === 'GET_GEOLOCATION') {
        navigator.geolocation.getCurrentPosition(
            (position) => {
                sendResponse({ position: position });
            },
            (error) => {
                sendResponse({ error: error.message });
            },
            request.options
        );
        // Return true to indicate we will respond asynchronously
        return true;
    }
});