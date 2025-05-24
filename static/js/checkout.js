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
    
    
    document.getElementById('checkoutForm').addEventListener('submit', function(event) {
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
        fetch('/api/v1/mpesa/checkout', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: jsonData
        })
        .then(response => {
            console.log(response)
            if (!response.ok) {
                // Handle HTTP errors (e.g., 500, 400)
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            return response.json(); // Parse the JSON response
        })
        .then(data => {
           console.log(data)
            // Handle the JSON response from your server
            if (data.ResponseCode === 0 || data.ResponseCode === '0') { //check for string
                // Redirect to the success page
              //  window.location.href = '/confirm?ip='+ip+"&redirect_url="+redirectUrl+"&devices="+quantity+"&phone="+phoneNumber; // important
                // Optionally reset the button if you navigate back later
                 payButton.disabled = false;
                 payButton.innerHTML = '<span>Pay Now</span><i class="fas fa-arrow-right"></i>';
            } else {
                // Handle M-Pesa business logic errors (e.g., insufficient funds)
                showAlert("Something went wrong, re-enter phone number!", "error"); // show to user
                // Re-enable the submit button and reset text
                payButton.disabled = false;
                payButton.innerHTML = '<span>Pay Now</span><i class="fas fa-arrow-right"></i>';
            }
        })
        .catch(error => {
            // Handle network errors or errors in the fetch() chain
            console.error('Error:', error);
            showAlert("Something went wrong, re-enter phone number!", "error");
            // Re-enable the submit button and reset text
            payButton.disabled = false;
            payButton.innerHTML = '<span>Pay Now</span><i class="fas fa-arrow-right"></i>';
        });
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