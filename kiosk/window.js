chrome.storage.local.get(['display', 'serial'], storage =>
	document.getElementsByTagName('webview')[0].src =
		`${storage.display}#${storage.serial}`
);
