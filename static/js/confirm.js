document.addEventListener('DOMContentLoaded', function() {
    // Get URL params
    const urlParams = new URLSearchParams(window.location.search);
    const clientIP = urlParams.get('ip');
    const redirectUrl = urlParams.get('redirect_url');
    
    // Elements
    const paymentStatus = document.getElementById('payment-status');
    const statusIcon = paymentStatus.querySelector('.status-icon');
    const statusTitle = paymentStatus.querySelector('.status-title');
    const statusMessage = paymentStatus.querySelector('.status-message');
    const connectButton = document.getElementById('connect-button');
    const paymentResult = document.getElementById('payment-result');
    const receiptNumber = document.getElementById('receipt-number');
    const transactionTime = document.getElementById('transaction-time');
    const transactionDetails = document.getElementById('transaction-details');
    const tryAgainSection = document.getElementById('try-again-section');
    const countdown = document.getElementById('countdown');
    const timerSection = document.getElementById('timer-section');
    const receiptRow = document.getElementById('receipt-row');
    
    // Show current date and time
    const now = new Date();
    transactionTime.textContent = now.toLocaleString();
    
    // Initialize WebSocket connection
    let socket;
    let countdownInterval;
    let timeLeft = 60; // 2 minutes countdown
    
    function connectWebSocket() {
        // Use secure WebSocket if the page is loaded over HTTPS
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        socket = new WebSocket(`${protocol}//${host}/ws?ip=${clientIP}`);
        
        socket.onopen = function() {
            console.log('WebSocket connected');
            // Show payment pending UI after connection is established
            startCountdown();
            // After 30 seconds of waiting, show the try again option
            setTimeout(function() {
                if (paymentStatus.classList.contains('pending')) {
                    tryAgainSection.style.display = 'block';
                }
            }, 30000);
        };
        
        socket.onmessage = function(event) {
            const data = JSON.parse(event.data);
            console.log('WebSocket message received:', data);
            
            if (data.type === 'payment') {
                handlePaymentUpdate(data);
            }
        };
        
        socket.onclose = function() {
            console.log('WebSocket connection closed');
            // Try to reconnect after 5 seconds
            setTimeout(connectWebSocket, 5000);
        };
        
        socket.onerror = function(error) {
            console.error('WebSocket error:', error);
        };
    }
    
    function handlePaymentUpdate(data) {
        // Clear countdown timer
        clearInterval(countdownInterval);
        timerSection.classList.add('hidden');
        
        // Show transaction details
        transactionDetails.style.display = 'block';
        
        // Update UI based on payment status
        if (data.status === 'success') {
            // Success state
            paymentStatus.className = 'payment-status success fade-in';
            statusIcon.className = 'fas fa-check-circle status-icon';
            statusTitle.textContent = 'Payment Successful!';
            statusMessage.textContent = 'Your payment has been confirmed. You can now connect to Wi-Fi.';
            
            // Show receipt number if available
            if (data.receiptNumber) {
                receiptNumber.textContent = data.receiptNumber;
            } else {
                receiptRow.style.display = 'none';
            }
            
            paymentResult.textContent = 'Successful';
            
            // Show connect button and update it with redirect URL
            connectButton.classList.remove('hidden');
            connectButton.addEventListener('click', function() {
                window.location.href = redirectUrl;
            });
            
            // Hide try again section
            tryAgainSection.style.display = 'none';
            
            // Show success notification
            showAlert('Payment successful! You can now connect to Wi-Fi.', 'success');
            
            // Auto-redirect after 5 seconds
            setTimeout(function() {
                window.location.href = "/howto"; //redirectUrl for single device
            }, 5000);
        } else {
            // Error state
            paymentStatus.className = 'payment-status error fade-in';
            statusIcon.className = 'fas fa-times-circle status-icon';
            statusTitle.textContent = 'Payment Failed';
            statusMessage.textContent = data.message || 'There was an issue with your payment.';
            
            paymentResult.textContent = 'Failed';
            receiptRow.style.display = 'none';
            
            // Show try again section
            tryAgainSection.style.display = 'block';
            
            // Show error notification
            showAlert(data.message || 'Payment failed. Please try again.', 'error');
        }
    }
    
    function startCountdown() {
        countdownInterval = setInterval(function() {
            timeLeft--;
            countdown.textContent = timeLeft;
            
            if (timeLeft <= 0) {
                clearInterval(countdownInterval);
                
                // If we haven't received a WebSocket message, show payment timeout
                if (paymentStatus.classList.contains('pending')) {
                    paymentStatus.className = 'payment-status error fade-in';
                    statusIcon.className = 'fas fa-clock status-icon';
                    statusTitle.textContent = 'Payment Timeout';
                    statusMessage.textContent = 'We haven\'t received your payment confirmation. Please try again.';
                    
                    // Show transaction details with timeout info
                    transactionDetails.style.display = 'block';
                    paymentResult.textContent = 'Timeout';
                    receiptRow.style.display = 'none';
                    
                    // Show try again section
                    tryAgainSection.style.display = 'block';
                    
                    // Hide timer section
                    timerSection.classList.add('hidden');
                }
            }
        }, 1000);
    }
    
    // Function to show alerts
    function showAlert(message, type) {
        const alertContainer = document.getElementById('alert-container');
        const alertClass = type === 'error' ? 'alert-error' : 'alert-success';
        const icon = type === 'error' ? 'exclamation-circle' : 'check-circle';
        
        alertContainer.innerHTML = `
            <div class="alert ${alertClass}">
                <i class="fas fa-${icon}"></i>
                ${message}
            </div>
        `;
        
        // Auto-remove after 5 seconds
        setTimeout(function() {
            alertContainer.innerHTML = '';
        }, 5000);
    }
    
    // Try again button handler
    document.getElementById('try-again-button').addEventListener('click', function() {
        window.location.href = document.referrer || '/';
    });
    
    // Back button handler
    document.getElementById('back-button').addEventListener('click', function(e) {
        e.preventDefault();
        window.location.href = redirectUrl || document.referrer || '/';
    });
    
    // Connect to WebSocket on page load
    connectWebSocket();
});