// MongoDB seed script for Holy Home application - CORRECTED VERSION
// Run with: docker-compose exec -T mongo mongosh holyhome < seed_data_fixed.js

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

function dec(value) {
    return NumberDecimal(value.toFixed(3));
}

function addMonths(date, months) {
    const result = new Date(date);
    result.setMonth(result.getMonth() + months);
    return result;
}

function getMonthDates(year, month) {
    const start = new Date(year, month - 1, 1);
    const end = new Date(year, month, 0, 23, 59, 59);
    return {start, end};
}

// Create groups
print("\n=== Creating Groups ===");
const groups = [
    {
        _id: ObjectId(),
        name: "Wspólne części",
        weight: 1.0,
        created_at: new Date("2024-01-01")
    },
    {
        _id: ObjectId(),
        name: "Garaż",
        weight: 0.5,
        created_at: new Date("2024-01-01")
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
let prevElecReading1 = 15000;
let prevElecReading2 = 12000;

for (let d = new Date(startDate); d < endDate; d = addMonths(d, 1)) {
    const year = d.getFullYear();
    const month = d.getMonth() + 1;
    const dates = getMonthDates(year, month);

    // Generate realistic consumption (200-400 kWh per person)
    const units1 = randomFloat(200, 400);
    const units2 = randomFloat(200, 400);
    const currentReading1 = prevElecReading1 + units1;
    const currentReading2 = prevElecReading2 + units2;

    const totalUnits = units1 + units2;
    const pricePerUnit = randomFloat(0.68, 0.72);
    const totalCost = totalUnits * pricePerUnit;

    const billDate = new Date(year, month - 1, 25);

    const bill = {
        _id: ObjectId(),
        type: "electricity",
        period_start: dates.start,
        period_end: dates.end,
        total_amount_pln: dec(totalCost),
        total_units: dec(totalUnits),
        status: "closed",
        created_at: billDate
    };

    db.bills.insertOne(bill);
    billCount++;

    // Create consumptions
    db.consumptions.insertMany([
        {
            _id: ObjectId(),
            bill_id: bill._id,
            user_id: user1._id,
            units: dec(units1),
            meter_value: dec(currentReading1),
            recorded_at: billDate,
            source: "user"
        },
        {
            _id: ObjectId(),
            bill_id: bill._id,
            user_id: user2._id,
            units: dec(units2),
            meter_value: dec(currentReading2),
            recorded_at: billDate,
            source: "user"
        }
    ]);
    consumptionCount += 2;

    // Create allocations (70% personal, 30% common)
    const personalCost = totalCost * 0.7;
    const commonCost = totalCost * 0.3;

    const user1PersonalCost = (units1 / totalUnits) * personalCost;
    const user2PersonalCost = (units2 / totalUnits) * personalCost;

    const totalWeight = groups.reduce((sum, g) => sum + g.weight, 0) + 2;
    const user1CommonCost = (1.0 / totalWeight) * commonCost;
    const user2CommonCost = (1.0 / totalWeight) * commonCost;

    const allocations = [
        {
            _id: ObjectId(),
            bill_id: bill._id,
            subject_type: "user",
            subject_id: user1._id,
            amount_pln: dec(user1PersonalCost + user1CommonCost),
            units: dec(units1),
            method: "proportional"
        },
        {
            _id: ObjectId(),
            bill_id: bill._id,
            subject_type: "user",
            subject_id: user2._id,
            amount_pln: dec(user2PersonalCost + user2CommonCost),
            units: dec(units2),
            method: "proportional"
        }
    ];

    // Group allocations
    groups.forEach(group => {
        const groupCost = (group.weight / totalWeight) * commonCost;
        allocations.push({
            _id: ObjectId(),
            bill_id: bill._id,
            subject_type: "group",
            subject_id: group._id,
            amount_pln: dec(groupCost),
            units: dec(0),
            method: "weight"
        });
    });

    db.allocations.insertMany(allocations);
    allocationCount += allocations.length;

    prevElecReading1 = currentReading1;
    prevElecReading2 = currentReading2;
}

// Generate monthly gas bills
print("Generating gas bills...");
let prevGasReading1 = 5000;
let prevGasReading2 = 4500;

for (let d = new Date(startDate); d < endDate; d = addMonths(d, 1)) {
    const year = d.getFullYear();
    const month = d.getMonth() + 1;
    const dates = getMonthDates(year, month);

    // Gas varies by season (more in winter)
    const isWinter = month >= 11 || month <= 3;
    const baseUnits = isWinter ? randomFloat(80, 150) : randomFloat(30, 60);

    const units1 = baseUnits * randomFloat(0.9, 1.1);
    const units2 = baseUnits * randomFloat(0.9, 1.1);
    const currentReading1 = prevGasReading1 + units1;
    const currentReading2 = prevGasReading2 + units2;

    const totalUnits = units1 + units2;
    const pricePerUnit = randomFloat(0.35, 0.40);
    const totalCost = totalUnits * pricePerUnit;

    const billDate = new Date(year, month - 1, 20);

    const bill = {
        _id: ObjectId(),
        type: "gas",
        period_start: dates.start,
        period_end: dates.end,
        total_amount_pln: dec(totalCost),
        total_units: dec(totalUnits),
        status: "closed",
        created_at: billDate
    };

    db.bills.insertOne(bill);
    billCount++;

    // Consumptions
    db.consumptions.insertMany([
        {
            _id: ObjectId(),
            bill_id: bill._id,
            user_id: user1._id,
            units: dec(units1),
            meter_value: dec(currentReading1),
            recorded_at: billDate,
            source: "user"
        },
        {
            _id: ObjectId(),
            bill_id: bill._id,
            user_id: user2._id,
            units: dec(units2),
            meter_value: dec(currentReading2),
            recorded_at: billDate,
            source: "user"
        }
    ]);
    consumptionCount += 2;

    // Allocations (50-50 split)
    db.allocations.insertMany([
        {
            _id: ObjectId(),
            bill_id: bill._id,
            subject_type: "user",
            subject_id: user1._id,
            amount_pln: dec(totalCost * 0.5),
            units: dec(units1),
            method: "proportional"
        },
        {
            _id: ObjectId(),
            bill_id: bill._id,
            subject_type: "user",
            subject_id: user2._id,
            amount_pln: dec(totalCost * 0.5),
            units: dec(units2),
            method: "proportional"
        }
    ]);
    allocationCount += 2;

    prevGasReading1 = currentReading1;
    prevGasReading2 = currentReading2;
}

// Generate monthly internet bills
print("Generating internet bills...");
const internetCost = 99.99;

for (let d = new Date(startDate); d < endDate; d = addMonths(d, 1)) {
    const year = d.getFullYear();
    const month = d.getMonth() + 1;
    const dates = getMonthDates(year, month);
    const billDate = new Date(year, month - 1, 5);

    const bill = {
        _id: ObjectId(),
        type: "internet",
        period_start: dates.start,
        period_end: dates.end,
        total_amount_pln: dec(internetCost),
        status: "closed",
        created_at: billDate
    };

    db.bills.insertOne(bill);
    billCount++;

    // Equal split
    db.allocations.insertMany([
        {
            _id: ObjectId(),
            bill_id: bill._id,
            subject_type: "user",
            subject_id: user1._id,
            amount_pln: dec(internetCost * 0.5),
            units: dec(0),
            method: "equal"
        },
        {
            _id: ObjectId(),
            bill_id: bill._id,
            subject_type: "user",
            subject_id: user2._id,
            amount_pln: dec(internetCost * 0.5),
            units: dec(0),
            method: "equal"
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

const loanNotes = ["Za zakupy", "Paliwo", "Rachunek", "Pizza", "Kino", "Kosmetyki"];
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
    const isUser1Lender = idx % 2 === 0;
    const lenderId = isUser1Lender ? user1._id : user2._id;
    const borrowerId = isUser1Lender ? user2._id : user1._id;

    const amount = randomFloat(50, 500);
    const loan = {
        _id: ObjectId(),
        lender_id: lenderId,
        borrower_id: borrowerId,
        amount_pln: dec(amount),
        note: loanNotes[randomInt(0, loanNotes.length - 1)],
        status: "open",
        created_at: date
    };

    db.loans.insertOne(loan);
    loanCount++;

    // Some loans have partial/full payments
    if (Math.random() > 0.3) {
        const numPayments = randomInt(1, 3);
        let remaining = amount;

        for (let p = 0; p < numPayments && remaining > 5; p++) {
            const paymentAmount = p === numPayments - 1 ? remaining : remaining * randomFloat(0.3, 0.6);
            const paymentDate = addMonths(date, p + 1);

            if (paymentDate < endDate) {
                db.loan_payments.insertOne({
                    _id: ObjectId(),
                    loan_id: loan._id,
                    amount_pln: dec(paymentAmount),
                    paid_at: paymentDate,
                    created_at: paymentDate
                });

                remaining -= paymentAmount;
                paymentCount++;
            }
        }

        // Update status
        if (remaining < 0.01) {
            db.loans.updateOne({_id: loan._id}, {$set: {status: "settled"}});
        } else if (remaining < amount) {
            db.loans.updateOne({_id: loan._id}, {$set: {status: "partial"}});
        }
    }
});

print(`Created ${loanCount} loans`);
print(`Created ${paymentCount} loan payments`);

// Summary
print("\n=== SEED DATA SUMMARY ===");
print(`Groups: ${groups.length}`);
print(`Bills: ${billCount}`);
print(`Consumptions: ${consumptionCount}`);
print(`Allocations: ${allocationCount}`);
print(`Loans: ${loanCount}`);
print(`Loan Payments: ${paymentCount}`);
print("\n✅ Seed data generation complete!");
