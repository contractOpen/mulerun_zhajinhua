const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("ZhajinhuaChips", function () {
  async function deployContract() {
    const fallbackPriceUsd = 2000n * 10n ** 8n;
    const stalenessThreshold = 3600;
    const [owner, buyer, treasury] = await ethers.getSigners();

    const ZhajinhuaChips = await ethers.getContractFactory("ZhajinhuaChips");
    const contract = await ZhajinhuaChips.deploy(owner.address, fallbackPriceUsd, stalenessThreshold);
    await contract.waitForDeployment();

    return { contract, owner, buyer, treasury, fallbackPriceUsd };
  }

  it("stores constructor values", async function () {
    const { contract, owner, fallbackPriceUsd } = await deployContract();

    expect(await contract.owner()).to.equal(owner.address);
    expect(await contract.fallbackPriceUsd()).to.equal(fallbackPriceUsd);
  });

  it("awards points on buyChips", async function () {
    const { contract, buyer } = await deployContract();
    const value = ethers.parseEther("0.1");

    await expect(contract.connect(buyer).buyChips({ value }))
      .to.emit(contract, "ChipsPurchased");

    expect(await contract.totalDeposited(buyer.address)).to.equal(value);
    expect(await contract.totalPoints(buyer.address)).to.equal(200000n);
  });

  it("lets owner withdraw funds", async function () {
    const { contract, buyer, treasury } = await deployContract();
    const value = ethers.parseEther("0.1");

    await contract.connect(buyer).buyChips({ value });

    await expect(contract.withdraw(treasury.address))
      .to.emit(contract, "FundsWithdrawn");

    expect(await ethers.provider.getBalance(await contract.getAddress())).to.equal(0n);
  });
});
