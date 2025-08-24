import os
from fastapi import FastAPI, HTTPException, Depends
from sqlalchemy import create_engine, Column, Integer, String, Numeric, DateTime, ForeignKey
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker, Session
from pydantic import BaseModel
import httpx
from datetime import datetime
from prometheus_fastapi_instrumentator import Instrumentator

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql+psycopg2://order_user:order_pass@postgres:5432/order_service")
engine = create_engine(DATABASE_URL)
SessionLocal = sessionmaker(bind=engine, expire_on_commit=False)
Base = declarative_base()

class Order(Base):
    __tablename__ = "orders"
    id = Column(Integer, primary_key=True, index=True)
    user_id = Column(Integer, nullable=False)
    product_id = Column(Integer, nullable=False)
    quantity = Column(Integer, nullable=False)
    total = Column(Numeric(10,2), nullable=False)
    status = Column(String, default="PENDING")
    created_at = Column(DateTime, default=datetime.utcnow)

Base.metadata.create_all(bind=engine)

app = FastAPI(title="Order Service")
Instrumentator().instrument(app).expose(app)

def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

class OrderCreate(BaseModel):
    user_id: int
    product_id: int
    quantity: int

@app.get("/healthz")
def healthz():
    return "ok"

@app.post("/orders")
def create_order(order: OrderCreate, db: Session = Depends(get_db)):
    # fetch product price
    r = httpx.get(f"http://product-catalog:8080/products/{order.product_id}")
    if r.status_code != 200:
        raise HTTPException(status_code=400, detail="product not found")
    product = r.json()
    total = float(product["price"]) * order.quantity
    if product["stock"] < order.quantity:
        raise HTTPException(status_code=400, detail="insufficient stock")

    db_order = Order(user_id=order.user_id, product_id=order.product_id,
                     quantity=order.quantity, total=total)
    db.add(db_order)
    db.commit()
    db.refresh(db_order)
    return db_order

@app.get("/orders/{order_id}")
def get_order(order_id: int, db: Session = Depends(get_db)):
    order = db.query(Order).filter(Order.id == order_id).first()
    if not order:
        raise HTTPException(status_code=404, detail="not found")
    return order