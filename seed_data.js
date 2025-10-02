// MongoDB seed script for Holy Home application
// Run with: docker-compose exec -T mongo mongosh holyhome < seed_data.js

// Get existing users
const existingUsers = db.users.find().toArray();
print(`Found ${existingUsers.length} existing users`);

if (existingUsers.length < 2) {
    print("ERROR: Need at least 2 users to generate meaningful data");
    quit(1);
}

const user1 = existingUsers[0];
const user2 = existingUsers[1];
print(`User 1: ${user1.name} (${user1.email})`);
print(`User 2: ${user2.name} (${user2.email})`);

// Helper functions
function randomFloat(min, max) {
    return Math.random() * (max - min) + min;
}

function randomInt(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

function decimalFromFloat(value) {
    return NumberDecimal(value.toFixed(2));
}

function getMonthPeriod(year, month) {
    const monthStr = month.toString().padStart(2, '0');
    return `${year}-${monthStr}`;
}

function addMonths(date, months) {
    const result = new Date(date);
    result.setMonth(result.getMonth() + months);
    return result;
}

// Create groups
print("\n=== Creating Groups ===");
const groups = [
    {
        _id: ObjectId(),
        name: "Wspólne części",
        weight: 1.0,
        createdAt: new Date("2024-01-01")
    },
    {
        _id: ObjectId(),
        name: "Garaż",
        weight: 0.5,
        createdAt: new Date("2024-01-01")
    }
];

db.groups.insertMany(groups);
print(`Created ${groups.length} groups`);

// Generate bills and consumptions for the past year
print("\n=== Generating Bills and Consumptions ===");

const startDate = new Date("2024-01-01");
const endDate = new Date("2025-01-01");

let billCount = 0;
let consumptionCount = 0;
let allocationCount = 0;

// Generate monthly electricity bills
print("\nGenerating electricity bills...");
let prevElectricityReading1 = 15000;
let prevElectricityReading2 = 12000;

for (let d = new Date(startDate); d < endDate; d = addMonths(d, 1)) {
    const year = d.getFullYear();
    const month = d.getMonth() + 1;
    const period = getMonthPeriod(year, month);

    // Generate realistic consumption (between 200-400 kWh per person per month)
    const consumption1 = randomFloat(200, 400);
    const consumption2 = randomFloat(200, 400);
    const currentReading1 = prevElectricityReading1 + consumption1;
    const currentReading2 = prevElectricityReading2 + consumption2;

    const totalUnits = consumption1 + consumption2;
    // Price per kWh around 0.70 PLN
    const pricePerUnit = randomFloat(0.68, 0.72);
    const totalCost = totalUnits * pricePerUnit;

    const billDate = new Date(year, month - 1, 25);

    const electricityBill = {
        _id: ObjectId(),
        type: "electricity",
        period: period,
        amountPLN: decimalFromFloat(totalCost),
        totalUnits: decimalFromFloat(totalUnits),
        unitPrice: decimalFromFloat(pricePerUnit),
        status: "closed",
        closedAt: billDate,
        createdAt: billDate
    };

    db.bills.insertOne(electricityBill);
    billCount++;

    // Create consumptions
    const consumption1Doc = {
        _id: ObjectId(),
        billId: electricityBill._id,
        userId: user1._id,
        previousReading: decimalFromFloat(prevElectricityReading1),
        currentReading: decimalFromFloat(currentReading1),
        unitsConsumed: decimalFromFloat(consumption1),
        readingDate: billDate,
        createdAt: billDate
    };

    const consumption2Doc = {
        _id: ObjectId(),
        billId: electricityBill._id,
        userId: user2._id,
        previousReading: decimalFromFloat(prevElectricityReading2),
        currentReading: decimalFromFloat(currentReading2),
        unitsConsumed: decimalFromFloat(consumption2),
        readingDate: billDate,
        createdAt: billDate
    };

    db.consumptions.insertMany([consumption1Doc, consumption2Doc]);
    consumptionCount += 2;

    // Create allocations (personal + common area split)
    const personalPoolSize = 0.7; // 70% personal, 30% common
    const personalCost = totalCost * personalPoolSize;
    const commonCost = totalCost * (1 - personalPoolSize);

    const user1PersonalCost = (consumption1 / totalUnits) * personalCost;
    const user2PersonalCost = (consumption2 / totalUnits) * personalCost;

    const totalWeight = groups.reduce((sum, g) => sum + g.weight, 0) + 2; // 2 users + groups
    const user1CommonCost = (1.0 / totalWeight) * commonCost;
    const user2CommonCost = (1.0 / totalWeight) * commonCost;

    const allocations = [
        {
            _id: ObjectId(),
            billId: electricityBill._id,
            subjectType: "user",
            subjectId: user1._id,
            amountPLN: decimalFromFloat(user1PersonalCost + user1CommonCost),
            createdAt: billDate
        },
        {
            _id: ObjectId(),
            billId: electricityBill._id,
            subjectType: "user",
            subjectId: user2._id,
            amountPLN: decimalFromFloat(user2PersonalCost + user2CommonCost),
            createdAt: billDate
        }
    ];

    // Add group allocations
    groups.forEach(group => {
        const groupCost = (group.weight / totalWeight) * commonCost;
        allocations.push({
            _id: ObjectId(),
            billId: electricityBill._id,
            subjectType: "group",
            subjectId: group._id,
            amountPLN: decimalFromFloat(groupCost),
            createdAt: billDate
        });
    });

    db.allocations.insertMany(allocations);
    allocationCount += allocations.length;

    prevElectricityReading1 = currentReading1;
    prevElectricityReading2 = currentReading2;
}

// Generate monthly gas bills
print("Generating gas bills...");
let prevGasReading1 = 5000;
let prevGasReading2 = 4500;

for (let d = new Date(startDate); d < endDate; d = addMonths(d, 1)) {
    const year = d.getFullYear();
    const month = d.getMonth() + 1;
    const period = getMonthPeriod(year, month);

    // Gas consumption varies by season (more in winter)
    const isWinter = month >= 11 || month <= 3;
    const baseConsumption = isWinter ? randomFloat(80, 150) : randomFloat(30, 60);

    const consumption1 = baseConsumption * randomFloat(0.9, 1.1);
    const consumption2 = baseConsumption * randomFloat(0.9, 1.1);
    const currentReading1 = prevGasReading1 + consumption1;
    const currentReading2 = prevGasReading2 + consumption2;

    const totalUnits = consumption1 + consumption2;
    const pricePerUnit = randomFloat(0.35, 0.40);
    const totalCost = totalUnits * pricePerUnit;

    const billDate = new Date(year, month - 1, 20);

    const gasBill = {
        _id: ObjectId(),
        type: "gas",
        period: period,
        amountPLN: decimalFromFloat(totalCost),
        totalUnits: decimalFromFloat(totalUnits),
        unitPrice: decimalFromFloat(pricePerUnit),
        status: "closed",
        closedAt: billDate,
        createdAt: billDate
    };

    db.bills.insertOne(gasBill);
    billCount++;

    // Create consumptions
    db.consumptions.insertMany([
        {
            _id: ObjectId(),
            billId: gasBill._id,
            userId: user1._id,
            previousReading: decimalFromFloat(prevGasReading1),
            currentReading: decimalFromFloat(currentReading1),
            unitsConsumed: decimalFromFloat(consumption1),
            readingDate: billDate,
            createdAt: billDate
        },
        {
            _id: ObjectId(),
            billId: gasBill._id,
            userId: user2._id,
            previousReading: decimalFromFloat(prevGasReading2),
            currentReading: decimalFromFloat(currentReading2),
            unitsConsumed: decimalFromFloat(consumption2),
            readingDate: billDate,
            createdAt: billDate
        }
    ]);
    consumptionCount += 2;

    // Allocations (50-50 split for gas)
    db.allocations.insertMany([
        {
            _id: ObjectId(),
            billId: gasBill._id,
            subjectType: "user",
            subjectId: user1._id,
            amountPLN: decimalFromFloat(totalCost * 0.5),
            createdAt: billDate
        },
        {
            _id: ObjectId(),
            billId: gasBill._id,
            subjectType: "user",
            subjectId: user2._id,
            amountPLN: decimalFromFloat(totalCost * 0.5),
            createdAt: billDate
        }
    ]);
    allocationCount += 2;

    prevGasReading1 = currentReading1;
    prevGasReading2 = currentReading2;
}

// Generate monthly internet bills
print("Generating internet bills...");
const internetCost = 99.99; // Fixed cost

for (let d = new Date(startDate); d < endDate; d = addMonths(d, 1)) {
    const year = d.getFullYear();
    const month = d.getMonth() + 1;
    const period = getMonthPeriod(year, month);
    const billDate = new Date(year, month - 1, 5);

    const internetBill = {
        _id: ObjectId(),
        type: "internet",
        period: period,
        amountPLN: decimalFromFloat(internetCost),
        status: "closed",
        closedAt: billDate,
        createdAt: billDate
    };

    db.bills.insertOne(internetBill);
    billCount++;

    // Equal split for internet
    db.allocations.insertMany([
        {
            _id: ObjectId(),
            billId: internetBill._id,
            subjectType: "user",
            subjectId: user1._id,
            amountPLN: decimalFromFloat(internetCost * 0.5),
            createdAt: billDate
        },
        {
            _id: ObjectId(),
            billId: internetBill._id,
            subjectType: "user",
            subjectId: user2._id,
            amountPLN: decimalFromFloat(internetCost * 0.5),
            createdAt: billDate
        }
    ]);
    allocationCount += 2;
}

print(`Created ${billCount} bills`);
print(`Created ${consumptionCount} consumptions`);
print(`Created ${allocationCount} allocations`);

// Generate loan history
print("\n=== Generating Loan History ===");
let loanCount = 0;
let paymentCount = 0;

// Generate some random loans throughout the year
const loanDates = [
    new Date("2024-02-10"),
    new Date("2024-03-15"),
    new Date("2024-05-20"),
    new Date("2024-06-05"),
    new Date("2024-08-12"),
    new Date("2024-09-18"),
    new Date("2024-11-03")
];

loanDates.forEach((date, idx) => {
    // Alternate lender/borrower
    const isUser1Lender = idx % 2 === 0;
    const lenderId = isUser1Lender ? user1._id : user2._id;
    const borrowerId = isUser1Lender ? user2._id : user1._id;

    const amount = randomFloat(50, 500);
    const loan = {
        _id: ObjectId(),
        lenderId: lenderId,
        borrowerId: borrowerId,
        amountPLN: decimalFromFloat(amount),
        note: ["Za zakupy", "Paliwo", "Rachunek za prąd", "Pizza", "Kino"][randomInt(0, 4)],
        status: "open",
        createdAt: date
    };

    db.loans.insertOne(loan);
    loanCount++;

    // Some loans have partial payments
    if (Math.random() > 0.3) {
        const numPayments = randomInt(1, 3);
        let remainingAmount = amount;

        for (let p = 0; p < numPayments && remainingAmount > 5; p++) {
            const paymentAmount = p === numPayments - 1 ? remainingAmount : remainingAmount * randomFloat(0.3, 0.6);
            const paymentDate = addMonths(date, p + 1);

            if (paymentDate < endDate) {
                db.loan_payments.insertOne({
                    _id: ObjectId(),
                    loanId: loan._id,
                    amountPLN: decimalFromFloat(paymentAmount),
                    paidAt: paymentDate,
                    createdAt: paymentDate
                });

                remainingAmount -= paymentAmount;
                paymentCount++;
            }
        }

        // Update loan status
        if (remainingAmount < 0.01) {
            db.loans.updateOne({_id: loan._id}, {$set: {status: "settled"}});
        } else if (remainingAmount < amount) {
            db.loans.updateOne({_id: loan._id}, {$set: {status: "partial"}});
        }
    }
});

print(`Created ${loanCount} loans`);
print(`Created ${paymentCount} loan payments`);

// Summary
print("\n=== Seed Data Summary ===");
print(`Groups: ${groups.length}`);
print(`Bills: ${billCount}`);
print(`Consumptions: ${consumptionCount}`);
print(`Allocations: ${allocationCount}`);
print(`Loans: ${loanCount}`);
print(`Loan Payments: ${paymentCount}`);
print("\n✅ Seed data generation complete!");
