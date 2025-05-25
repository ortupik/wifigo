document.addEventListener('DOMContentLoaded', function() {
    const decreaseBtn = document.getElementById('decrease-btn');
    const increaseBtn = document.getElementById('increase-btn');
    const quantityInput = document.getElementById('quantity');
    const unitPriceEl = document.getElementById('unit-price');
    const calculationEl = document.getElementById('price-calculation');
    const totalEl = document.getElementById('total');
    const totalAmountInput = document.getElementById('total_amount');
    const deviceCountInput = document.getElementById('devices');
    const payButton = document.getElementById('payButton');
    
    // Get unit price from the displayed text
    const unitPriceText = unitPriceEl.textContent;
    const unitPrice = parseFloat(unitPriceText.replace('KES ', ''));
    const discountRowEl = document.getElementById('discount-row');
    const discountAmountEl = document.getElementById('discount-amount');

    function toggleInfo(elementId) {
        const infoBox = document.getElementById(elementId);
        infoBox.classList.toggle('hidden');
      }
    
    // Update price calculations
    function updatePrice() {
        const quantity = parseInt(quantityInput.value);
        let total = quantity * unitPrice;
        let discountAmount = 0;

        // Apply discount for 2 or more devices
        if (quantity >= 2) {
            const discountRate = 0.30; // 30% discount
            discountAmount = total * discountRate;
            total -= discountAmount;
            discountRowEl.style.display = 'flex'; // Show the discount row
            discountAmountEl.textContent = `-KES ${discountAmount.toFixed(2)}`;
        } else {
            discountRowEl.style.display = 'none'; // Hide the discount row
            discountAmountEl.textContent = `-KES 0.00`;
        }

        calculationEl.textContent = `${quantity} × KES ${unitPrice.toFixed(2)}`;
        totalEl.textContent = `KES ${total.toFixed(2)}`;

        // Update hidden fields
        totalAmountInput.value = total.toFixed(2);
        deviceCountInput.value = quantity;
    }
    
    // Event listeners for device quantity buttons
    decreaseBtn.addEventListener('click', function() {
        const currentValue = parseInt(quantityInput.value);
        if (currentValue > 1) {
            quantityInput.value = currentValue - 1;
            updatePrice();
        }
    });
    
    increaseBtn.addEventListener('click', function() {
        const currentValue = parseInt(quantityInput.value);
        if (currentValue < 10) {
            quantityInput.value = currentValue + 1;
            updatePrice();
        }
    });
    

    document.getElementById('checkoutForm').addEventListener('submit', async function(event) {
        event.preventDefault(); // Prevent the default form submission
    
        const phoneNumber = document.getElementById('phone').value;
        const quantity = document.getElementById('quantity').value; 
        const planId = document.getElementById('plan_id').value;
        const deviceId = document.getElementById('device_id').value;
        const zone = document.getElementById('zone').value;
        const mac = document.getElementById('mac').value;
        const ip = document.getElementById('ip').value;
        const dns_name =  document.getElementById('dns_name').value;
        const ispId = document.getElementById('isp_id').value;
        const redirectUrl = "http://"+zone+"."+dns_name;
    
        // Simple validation
        if (!phoneNumber || phoneNumber.length < 10) {
            showAlert('Please enter a valid phone number', 'error');
            return;
        }
    
        // Show loading state
        payButton.disabled = true;
        payButton.innerHTML = '<span>Processing...</span><div class="spinner"></div>';
          
        // Get form data
        const formDataObject = {
            isp_id: ispId,
            phone: phoneNumber,
            plan_id: parseInt(planId, 10), 
            device_id: deviceId,
            zone: zone,
            devices: parseInt(quantity, 10), 
            mac: mac,
            ip: ip,
            dns_name: dns_name
        };

    
        // Convert form data to JSON
        const jsonData = JSON.stringify(formDataObject);
    
        // Send the POST request to your API endpoint
        try {
            const response = await fetch('/api/v1/mpesa/checkout', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: jsonData
            });

            if (!response.ok) {
                // Attempt to parse the error response as JSON
                // This will be caught by the outer catch block if parsing fails
                const errorData = await response.json();
                
                // You can now access properties from errorData
                if (response.status === 409 && errorData.error === "Active subscription already exists") {
                    throw new Error("active_subscription"); // Throw a specific error identifier
                } else {
                    // For other HTTP errors (e.g., 400, 500)
                    throw new Error(errorData.error || "An unexpected error occurred.");
                }
            }

            const data = await response.json(); // Parse the successful JSON response

            // Handle the JSON response from your server for successful M-Pesa initiation
            if (data.ResponseCode === 0 || data.ResponseCode === '0') {
                // Redirect to the success page
                 window.location.href = '/confirm?ip='+ip+"&redirect_url="+redirectUrl+"&devices="+quantity+"&phone="+phoneNumber; // important
                showAlert("M-Pesa payment initiated. Check your phone!", "success");
            } else {
                // Handle M-Pesa business logic errors (e.g., invalid phone, internal M-Pesa error)
                // Your Go backend sends 'errorCode', 'errorMessage', 'requestId'
                const errorMessage = data.errorMessage || "Something went wrong with the M-Pesa request.";
                showAlert(errorMessage, "error");
                console.error("M-Pesa Response Error:", data);
            }

        } catch (error) {
            // Handle network errors or errors thrown in the try block
            console.error('Fetch Error:', error);

            if (error.message === "active_subscription") {
                showAlert("You already have an active subscription!", "info");
            } else if (error.message.includes("Invalid request")) { // Catch specific error from your backend
                showAlert("Invalid request. Please check your details.", "error");
            } else if (error.message.includes("Invalid plan")) {
                showAlert("The selected plan is invalid.", "error");
            } else if (error.message.includes("STK Push failed")) {
                showAlert("Failed to initiate M-Pesa STK Push. Please try again.", "error");
            }
            else {
                showAlert("Something went wrong, please try again!", "error");
            }
        } finally {
            // This block always executes, regardless of success or failure
            payButton.disabled = false;
            payButton.innerHTML = '<span>Pay Now</span><i class="fas fa-arrow-right"></i>';
        }
    });
    
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

});