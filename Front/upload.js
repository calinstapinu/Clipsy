// upload.js
document.addEventListener("DOMContentLoaded", function () {
    const dropArea = document.getElementById("drop-area");
    const fileElem = document.getElementById("fileElem");
    const fileNameDisplay = document.getElementById("file-name");

    // Prevent default behaviors for drag and drop event
    ["dragenter", "dragover", "dragleave", "drop"].forEach(eventName => {
        dropArea.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });

    function preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }

    // Highlight the drop area when an item is dragged over it
    ["dragenter", "dragover"].forEach(eventName => {
        dropArea.addEventListener(eventName, () => dropArea.classList.add("bg-gray-200"), false);
    });

    ["dragleave", "drop"].forEach(eventName => {
        dropArea.addEventListener(eventName, () => dropArea.classList.remove("bg-gray-200"), false);
    });

    // Handle dropped files
    dropArea.addEventListener("drop", handleDrop, false);

    function handleDrop(e) {
        const dt = e.dataTransfer;
        const files = dt.files;

        // Set the selected file to the hidden input
        if (files.length > 0) {
            fileElem.files = files; // Update the file input with the dropped files

            // Display the file name
            fileNameDisplay.textContent = `Selected file: ${files[0].name}`;
        }
    }

    // Click event to trigger file input when drop area is clicked
    dropArea.addEventListener('click', () => {
        fileElem.click();
    });

    // Event listener for file input change
    fileElem.addEventListener('change', () => {
        // Check if a file was selected
        if (fileElem.files.length > 0) {
            fileNameDisplay.textContent = `Selected file: ${fileElem.files[0].name}`; // Display the selected file name
        }
    });
});
